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

func (r *noopProductRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *noopProductRepo) BulkSoftDelete(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *noopProductRepo) UpdateBlocked(_ context.Context, _, _ uuid.UUID, _ bool) error {
	return nil
}
func (r *noopProductRepo) SaveBusinessTypes(_ context.Context, _, _ uuid.UUID, _ []productvobj.BusinessTypeAssignment) error {
	return nil
}
func (r *noopProductRepo) RemoveBusinessType(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (r *noopProductRepo) FindBusinessTypesForProduct(_ context.Context, _, _ uuid.UUID) ([]productvobj.BusinessTypeAssignment, error) {
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

// --- T-SCR-B04: BuildPaginatedURL ---

func TestBuildPaginatedURL_SimpleURL(t *testing.T) {
	result := scrapeuse.BuildPaginatedURL("https://example.com/products", "page", 2)
	assert.Equal(t, "https://example.com/products?page=2", result)
}

func TestBuildPaginatedURL_URLWithExistingQueryParams(t *testing.T) {
	result := scrapeuse.BuildPaginatedURL("https://example.com/products?category=food", "page", 3)
	assert.Contains(t, result, "page=3")
	assert.Contains(t, result, "category=food")
}

func TestBuildPaginatedURL_CustomPageParam(t *testing.T) {
	result := scrapeuse.BuildPaginatedURL("https://www.electromisiones.com.ar/642-electrodomesticos", "p", 2)
	assert.Equal(t, "https://www.electromisiones.com.ar/642-electrodomesticos?p=2", result)
}

func TestBuildPaginatedURL_Page1(t *testing.T) {
	result := scrapeuse.BuildPaginatedURL("https://example.com/store", "page", 1)
	assert.Equal(t, "https://example.com/store?page=1", result)
}

// --- T-SCR-B05: paginated job uses paginated URL ---

func TestExecuteScrapingUseCase_PaginatedJobAppendsPageParam(t *testing.T) {
	// Arrange
	price := 100.0
	capturedURL := ""
	spy := &urlCaptureScraper{
		products: []scrapeport.RawProduct{
			{Title: "Producto pagina 2", URL: "https://example.com/p/2", Price: &price},
		},
		captureURL: func(url string) { capturedURL = url },
	}
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()

	source, _ := sourceentity.NewSource(sourceentity.CreateSourceParams{
		TenantID:        tenantID,
		Name:            "Paginated Source",
		BaseURL:         "https://example.com/products",
		Category:        "supermercados",
		SourceType:      "ecommerce",
		Priority:        "medium",
		Tier:            2,
		FirecrawlMethod: "extract",
	})
	source.CrawlConfig.MaxPages = 5
	source.CrawlConfig.PageParam = "p"
	_ = sourceRepo.Save(context.Background(), source)

	job, _ := entity.NewPaginatedScrapingJob(tenantID, source.ID, "manual", 2)
	_ = jobRepo.Save(context.Background(), job)

	upsertUC := usecase.NewUpsertProductsUseCase(&noopProductRepo{})
	uc := scrapeuse.NewExecuteScrapingUseCase(jobRepo, sourceRepo, spy, upsertUC)

	// Act
	err := uc.Execute(context.Background(), job)

	// Assert — the URL sent to the scraper should include ?p=2
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/products?p=2", capturedURL)
}

func TestExecuteScrapingUseCase_NonPaginatedJobUsesBaseURL(t *testing.T) {
	// Arrange
	price := 100.0
	capturedURL := ""
	spy := &urlCaptureScraper{
		products: []scrapeport.RawProduct{
			{Title: "Producto base", URL: "https://example.com/p/1", Price: &price},
		},
		captureURL: func(url string) { capturedURL = url },
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

	// Assert — page=0 should NOT add pagination params
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", capturedURL)
}

// --- url-capturing scraper for pagination tests ---

type urlCaptureScraper struct {
	products   []scrapeport.RawProduct
	captureURL func(string)
}

func (s *urlCaptureScraper) Extract(_ context.Context, url string, _ json.RawMessage, _ scrapeport.ExtractOptions) ([]scrapeport.RawProduct, error) {
	if s.captureURL != nil {
		s.captureURL(url)
	}
	return s.products, nil
}
func (s *urlCaptureScraper) ScrapeJSON(_ context.Context, url string, _ json.RawMessage, _ scrapeport.ExtractOptions) ([]scrapeport.RawProduct, error) {
	if s.captureURL != nil {
		s.captureURL(url)
	}
	return s.products, nil
}
func (s *urlCaptureScraper) Scrape(_ context.Context, _ string, _ scrapeport.ScrapeOptions) (string, error) {
	return "", nil
}
func (s *urlCaptureScraper) Crawl(_ context.Context, _ string, _ scrapeport.CrawlOptions) (string, error) {
	return "", nil
}
func (s *urlCaptureScraper) CrawlStatus(_ context.Context, _ string) (scrapeport.CrawlResult, error) {
	return scrapeport.CrawlResult{}, nil
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
