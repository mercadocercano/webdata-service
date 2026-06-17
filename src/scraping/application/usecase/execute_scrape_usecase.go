package usecase

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
)

type ExecuteScrapingUseCase struct {
	jobRepo    scrapingport.ScrapingJobRepository
	sourceRepo sourceport.SourceRepository
	scraper    scrapingport.ScraperPort
	upsertUC   *productusecase.UpsertProductsUseCase
}

func NewExecuteScrapingUseCase(
	jobRepo scrapingport.ScrapingJobRepository,
	sourceRepo sourceport.SourceRepository,
	scraper scrapingport.ScraperPort,
	upsertUC *productusecase.UpsertProductsUseCase,
) *ExecuteScrapingUseCase {
	return &ExecuteScrapingUseCase{
		jobRepo:    jobRepo,
		sourceRepo: sourceRepo,
		scraper:    scraper,
		upsertUC:   upsertUC,
	}
}

func (uc *ExecuteScrapingUseCase) ClaimJob(ctx context.Context) (*entity.ScrapingJob, error) {
	return uc.jobRepo.ClaimPendingJob(ctx)
}

func (uc *ExecuteScrapingUseCase) Execute(ctx context.Context, job *entity.ScrapingJob) error {
	// ClaimPendingJob already transitions the job to running atomically
	// (SELECT FOR UPDATE SKIP LOCKED + UPDATE status='running' in one tx)
	// so no need to call job.Start() here.

	source, err := uc.sourceRepo.FindByID(ctx, job.TenantID, job.SourceID)
	if err != nil {
		_ = job.Fail("source not found")
		_ = uc.jobRepo.Update(ctx, job)
		return err
	}

	// Build the target URL, appending pagination params when the job targets a specific page.
	targetURL := source.BaseURL
	if job.Page > 0 {
		targetURL = BuildPaginatedURL(source.BaseURL, source.CrawlConfig.EffectivePageParam(), job.Page)
	}

	rawProducts, err := uc.fetchProducts(ctx, source, targetURL)
	if err != nil {
		reason := fmt.Sprintf("%s failed: %v", source.FirecrawlMethod, err)
		_ = job.Fail(reason)
		_ = uc.jobRepo.Update(ctx, job)
		_ = uc.sourceRepo.RecordJobResult(ctx, source.ID, false)
		return err
	}

	// Fill missing product URLs with the source's base URL as a fallback.
	// Extraction schemas may not always include individual product page URLs.
	for i := range rawProducts {
		if rawProducts[i].URL == "" {
			rawProducts[i].URL = source.BaseURL
		}
	}

	created, updated, upsertErr := uc.upsertUC.ExecuteDetailedWithFilter(ctx, job.TenantID, job.SourceID, &job.ID, rawProducts, source.ExcludedBrands)

	found := len(rawProducts)
	saved := created + updated
	if upsertErr != nil {
		_ = job.Fail(upsertErr.Error())
	} else {
		_ = job.Complete(found, saved)
	}

	_ = uc.jobRepo.Update(ctx, job)
	_ = uc.sourceRepo.RecordJobResult(ctx, source.ID, upsertErr == nil)
	return upsertErr
}

// fetchProducts dispatches to Extract, Scrape, or Crawl based on the source's FirecrawlMethod.
// The targetURL may include pagination query params when the job targets a specific page.
func (uc *ExecuteScrapingUseCase) fetchProducts(ctx context.Context, source *sourceentity.Source, targetURL string) ([]scrapingport.RawProduct, error) {
	switch source.FirecrawlMethod {
	case "extract", "":
		return uc.scraper.Extract(ctx, targetURL, source.ExtractionSchema.Raw(), scrapingport.ExtractOptions{
			Prompt: source.Prompt,
		})

	case "scrape":
		return uc.scraper.ScrapeJSON(ctx, targetURL, source.ExtractionSchema.Raw(), scrapingport.ExtractOptions{
			Prompt: source.Prompt,
		})

	case "crawl":
		// Async crawl polling + product extraction is not yet implemented.
		// Calling Firecrawl here would consume credits with no benefit.
		// Migrate source to firecrawl_method='extract' to use LLM-powered product extraction.
		return nil, fmt.Errorf("firecrawl_method 'crawl' is not yet supported for product extraction; migrate source %s to use 'extract' method", source.ID)

	default:
		return nil, fmt.Errorf("unknown firecrawl_method %q for source %s", source.FirecrawlMethod, source.ID)
	}
}

// BuildPaginatedURL appends a pagination query parameter to the given base URL.
// It handles URLs that already have query parameters.
func BuildPaginatedURL(baseURL, pageParam string, page int) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		// Fallback: simple string concatenation if URL parsing fails.
		sep := "?"
		if len(baseURL) > 0 && (baseURL[len(baseURL)-1] == '?' || baseURL[len(baseURL)-1] == '&') {
			sep = ""
		} else if strings.Contains(baseURL, "?") {
			sep = "&"
		}
		return baseURL + sep + pageParam + "=" + strconv.Itoa(page)
	}
	q := u.Query()
	q.Set(pageParam, strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String()
}
