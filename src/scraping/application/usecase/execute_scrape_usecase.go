package usecase

import (
	"context"
	"fmt"

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

	// Bug 4 fix: dispatch to the correct adapter method based on source.FirecrawlMethod.
	rawProducts, err := uc.fetchProducts(ctx, source)
	if err != nil {
		reason := fmt.Sprintf("%s failed: %v", source.FirecrawlMethod, err)
		_ = job.Fail(reason)
		_ = uc.jobRepo.Update(ctx, job)
		source.RecordFailure(reason)
		_ = uc.sourceRepo.Update(ctx, source)
		return err
	}

	created, updated, upsertErr := uc.upsertUC.ExecuteDetailed(ctx, job.TenantID, job.SourceID, &job.ID, rawProducts)

	found := len(rawProducts)
	saved := created + updated
	if upsertErr != nil {
		_ = job.Fail(upsertErr.Error())
		source.RecordFailure(upsertErr.Error())
	} else {
		_ = job.Complete(found, saved)
		source.RecordSuccess()
	}

	_ = uc.jobRepo.Update(ctx, job)
	_ = uc.sourceRepo.Update(ctx, source)
	return upsertErr
}

// fetchProducts dispatches to Extract, Scrape, or Crawl based on the source's FirecrawlMethod.
// Only "extract" (the default) returns structured products directly.
func (uc *ExecuteScrapingUseCase) fetchProducts(ctx context.Context, source *sourceentity.Source) ([]scrapingport.RawProduct, error) {
	switch source.FirecrawlMethod {
	case "extract", "":
		return uc.scraper.Extract(ctx, source.BaseURL, source.ExtractionSchema.Raw(), scrapingport.ExtractOptions{})

	case "scrape":
		_, err := uc.scraper.Scrape(ctx, source.BaseURL, scrapingport.ScrapeOptions{IncludeMarkdown: true})
		if err != nil {
			return nil, fmt.Errorf("scrape: %w", err)
		}
		// Scrape returns raw content; structured product extraction from HTML/Markdown
		// is not yet implemented — return empty product list.
		return nil, nil

	case "crawl":
		_, err := uc.scraper.Crawl(ctx, source.BaseURL, scrapingport.CrawlOptions{
			MaxDepth:    source.CrawlConfig.MaxDepth,
			URLPatterns: source.CrawlConfig.URLPatterns,
		})
		if err != nil {
			return nil, fmt.Errorf("crawl: %w", err)
		}
		// Crawl starts an async job; product extraction requires a separate
		// polling flow that is not yet implemented — return empty product list.
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown firecrawl_method %q for source %s", source.FirecrawlMethod, source.ID)
	}
}
