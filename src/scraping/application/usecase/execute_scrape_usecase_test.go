package usecase_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	productentity "github.com/mercadocercano/webdata-service/src/product/domain/entity"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
	productvobj "github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapeuse "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock scraper ---

type spyScraper struct {
	scrapeCalled     bool
	scrapeJSONCalled bool
	crawlCalled      bool
	products         []scrapeport.RawProduct
}

func (s *spyScraper) Extract(_ context.Context, _ string, _ json.RawMessage, _ scrapeport.ExtractOptions) ([]scrapeport.RawProduct, error) {
	return s.products, nil
}

func (s *spyScraper) ScrapeJSON(_ context.Context, _ string, _ json.RawMessage, _ scrapeport.ExtractOptions) ([]scrapeport.RawProduct, error) {
	s.scrapeJSONCalled = true
	return s.products, nil
}

func (s *spyScraper) Scrape(_ context.Context, _ string, _ scrapeport.ScrapeOptions) (string, error) {
	s.scrapeCalled = true
	return "html content", nil
}

func (s *spyScraper) Crawl(_ context.Context, _ string, _ scrapeport.CrawlOptions) (string, error) {
	s.crawlCalled = true
	return "crawl-job-id", nil
}

func (s *spyScraper) CrawlStatus(_ context.Context, _ string) (scrapeport.CrawlResult, error) {
	return scrapeport.CrawlResult{}, nil
}

// --- Mock product repo (sin-op para los tests de scraping) ---

type noopProductRepo struct{}

func (r *noopProductRepo) Upsert(_ context.Context, _ *productentity.ScrapedProduct) (bool, error) {
	return true, nil
}

func (r *noopProductRepo) FindByID(_ context.Context, _, _ uuid.UUID) (*productentity.ScrapedProduct, error) {
	return nil, nil
}

func (r *noopProductRepo) FindByContentHash(_ context.Context, _, _ uuid.UUID, _ productvobj.ContentHash) (*productentity.ScrapedProduct, error) {
	return nil, nil
}

func (r *noopProductRepo) FindAll(_ context.Context, _ uuid.UUID, _ productport.ProductFilter) ([]*productentity.ScrapedProduct, int, error) {
	return nil, 0, nil
}

func (r *noopProductRepo) SavePriceRecord(_ context.Context, _ productvobj.PriceRecord) error {
	return nil
}

func (r *noopProductRepo) FindPriceHistory(_ context.Context, _, _ uuid.UUID) ([]productvobj.PriceRecord, error) {
	return nil, nil
}

// --- Helper: build executeScrapingUseCase ---

func buildExecuteUC(scraper scrapeport.ScraperPort) *scrapeuse.ExecuteScrapingUseCase {
	upsertUC := usecase.NewUpsertProductsUseCase(&noopProductRepo{})
	return scrapeuse.NewExecuteScrapingUseCase(
		newMockJobRepo(),
		newMockSourceRepo(),
		scraper,
		upsertUC,
	)
}

func buildJobWithSource(sourceRepo *mockSourceRepo, tenantID uuid.UUID, method string) (*entity.ScrapingJob, *sourceentity.Source) {
	source, _ := sourceentity.NewSource(sourceentity.CreateSourceParams{
		TenantID:        tenantID,
		Name:            "Test Source",
		BaseURL:         "https://example.com",
		Category:        "supermercados",
		SourceType:      "ecommerce",
		Priority:        "medium",
		Tier:            2,
		FirecrawlMethod: method,
	})
	_ = sourceRepo.Save(context.Background(), source)

	job, _ := entity.NewScrapingJob(tenantID, source.ID, "manual")
	return job, source
}

// --- T-SCR-B01: crawl method falla sin llamar a Firecrawl (regresión MER-79) ---

func TestExecuteScrapingUseCase_CrawlMethodReturnsErrorWithoutCallingFirecrawl(t *testing.T) {
	// Arrange
	spy := &spyScraper{}
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()

	job, _ := buildJobWithSource(sourceRepo, tenantID, "crawl")
	_ = jobRepo.Save(context.Background(), job)

	upsertUC := usecase.NewUpsertProductsUseCase(&noopProductRepo{})
	uc := scrapeuse.NewExecuteScrapingUseCase(jobRepo, sourceRepo, spy, upsertUC)

	// Act
	err := uc.Execute(context.Background(), job)

	// Assert — debe fallar con un mensaje claro, SIN llamar a Firecrawl
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crawl")
	assert.False(t, spy.crawlCalled, "no debe llamar a Firecrawl cuando crawl no está implementado")
}

// --- T-SCR-B02: scrape method usa ScrapeJSON y retorna productos (MER-141) ---

func TestExecuteScrapingUseCase_ScrapeMethodCallsScrapeJSONAndSavesProducts(t *testing.T) {
	// Arrange
	price := 150000.0
	spy := &spyScraper{
		products: []scrapeport.RawProduct{
			{Title: "Heladera 300L", URL: "https://cetrogar.com.ar/heladera", Price: &price},
		},
	}
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()

	job, _ := buildJobWithSource(sourceRepo, tenantID, "scrape")
	_ = jobRepo.Save(context.Background(), job)

	upsertUC := usecase.NewUpsertProductsUseCase(&noopProductRepo{})
	uc := scrapeuse.NewExecuteScrapingUseCase(jobRepo, sourceRepo, spy, upsertUC)

	// Act
	err := uc.Execute(context.Background(), job)

	// Assert — scrape debe usar ScrapeJSON y completar sin error
	require.NoError(t, err)
	assert.True(t, spy.scrapeJSONCalled, "debe llamar a ScrapeJSON para method=scrape")
	assert.False(t, spy.scrapeCalled, "no debe llamar a Scrape (HTML) para extracción de productos")
}

// --- T-SCR-B03: extract method sigue funcionando normalmente ---

func TestExecuteScrapingUseCase_ExtractMethodCallsFirecrawlAndSavesProducts(t *testing.T) {
	// Arrange
	price := 100.0
	spy := &spyScraper{
		products: []scrapeport.RawProduct{
			{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price},
		},
	}
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()

	job, _ := buildJobWithSource(sourceRepo, tenantID, "extract")
	_ = jobRepo.Save(context.Background(), job)

	upsertUC := usecase.NewUpsertProductsUseCase(&noopProductRepo{})
	uc := scrapeuse.NewExecuteScrapingUseCase(jobRepo, sourceRepo, spy, upsertUC)

	// Act
	err := uc.Execute(context.Background(), job)

	// Assert — extract debe seguir funcionando
	require.NoError(t, err)
}
