package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/request"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock product repo ---

type mockProductRepo struct {
	products     map[string]*entity.ScrapedProduct
	priceHistory []value_object.PriceRecord
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{products: make(map[string]*entity.ScrapedProduct)}
}

func (m *mockProductRepo) Upsert(ctx context.Context, p *entity.ScrapedProduct) (bool, error) {
	_, exists := m.products[p.ID.String()]
	m.products[p.ID.String()] = p
	return !exists, nil
}

// failingProductRepo simulates a repo where Upsert always fails (e.g. FK violation).
type failingProductRepo struct{}

func (r *failingProductRepo) Upsert(_ context.Context, _ *entity.ScrapedProduct) (bool, error) {
	return false, errors.New("pq: insert or update on table \"webdata_products\" violates foreign key constraint")
}

func (r *failingProductRepo) FindByContentHash(_ context.Context, _, _ uuid.UUID, _ value_object.ContentHash) (*entity.ScrapedProduct, error) {
	return nil, errors.New("product not found")
}

func (r *failingProductRepo) FindByID(_ context.Context, _, _ uuid.UUID) (*entity.ScrapedProduct, error) {
	return nil, errors.New("product not found")
}

func (r *failingProductRepo) FindAll(_ context.Context, _ uuid.UUID, _ port.ProductFilter) ([]*entity.ScrapedProduct, int, error) {
	return nil, 0, nil
}

func (r *failingProductRepo) SavePriceRecord(_ context.Context, _ value_object.PriceRecord) error {
	return nil
}

func (r *failingProductRepo) FindPriceHistory(_ context.Context, _, _ uuid.UUID) ([]value_object.PriceRecord, error) {
	return nil, nil
}

func (m *mockProductRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapedProduct, error) {
	p, ok := m.products[id.String()]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (m *mockProductRepo) FindByContentHash(ctx context.Context, tenantID, sourceID uuid.UUID, hash value_object.ContentHash) (*entity.ScrapedProduct, error) {
	for _, p := range m.products {
		if p.ContentHash == hash {
			return p, nil
		}
	}
	return nil, errors.New("product not found")
}

func (m *mockProductRepo) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.ProductFilter) ([]*entity.ScrapedProduct, int, error) {
	var result []*entity.ScrapedProduct
	for _, p := range m.products {
		result = append(result, p)
	}
	return result, len(result), nil
}

func (m *mockProductRepo) SavePriceRecord(ctx context.Context, record value_object.PriceRecord) error {
	m.priceHistory = append(m.priceHistory, record)
	return nil
}

func (m *mockProductRepo) FindPriceHistory(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.PriceRecord, error) {
	var result []value_object.PriceRecord
	for _, r := range m.priceHistory {
		if r.ProductID == productID {
			result = append(result, r)
		}
	}
	return result, nil
}

// --- T-PRD-A01: ListProducts ---

func TestListProductsUseCase_FiltersAndPagination(t *testing.T) {
	repo := newMockProductRepo()
	tenantID := uuid.New()
	sourceID := uuid.New()

	for i := 0; i < 3; i++ {
		p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
			TenantID: tenantID, SourceID: sourceID,
			Title: "Product", URL: "https://x.com/" + uuid.New().String(),
			ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Product", uuid.New().String()),
		})
		_, _ = repo.Upsert(context.Background(), p)
	}

	uc := usecase.NewListProductsUseCase(repo)
	result, err := uc.Execute(context.Background(), tenantID, request.ListProductsRequest{Page: 1, PageSize: 10})

	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
}

// --- T-PRD-A02: UpsertProducts dedup + price history ---

func TestUpsertProductsUseCase_DedupByHash(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo)
	tenantID := uuid.New()
	sourceID := uuid.New()

	price1 := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price1},
	}

	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	assert.Equal(t, 1, saved)
	assert.Len(t, repo.products, 1)

	// Upsert same product again — should not create duplicate
	saved, err = uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	assert.Equal(t, 1, saved)
	assert.Len(t, repo.products, 1)
}

func TestUpsertProductsUseCase_PriceHistoryOnChange(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo)
	tenantID := uuid.New()
	sourceID := uuid.New()

	price1 := 100.0
	price2 := 120.0
	raw := []scrapeport.RawProduct{{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price1}}

	_, _ = uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	initialHistory := len(repo.priceHistory)

	raw[0].Price = &price2
	_, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	assert.Greater(t, len(repo.priceHistory), initialHistory)
}

// --- T-PRD-A04: UpsertProducts no cuenta productos cuando Upsert falla (regresión MER-78) ---

func TestUpsertProductsUseCase_ReturnsZeroWhenUpsertFails(t *testing.T) {
	// Arrange — repo que siempre falla en Upsert (simula FK violation)
	repo := &failingProductRepo{}
	uc := usecase.NewUpsertProductsUseCase(repo)
	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price},
	}

	// Act
	saved, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), nil, raw)

	// Assert — el use case no propaga el error de upsert, pero tampoco cuenta el producto como guardado
	require.NoError(t, err)
	assert.Equal(t, 0, saved, "no debe contar productos cuyo Upsert falló")
}

func TestUpsertProductsUseCase_SkipsProductsWithEmptyTitle(t *testing.T) {
	// Arrange
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo)
	raw := []scrapeport.RawProduct{
		{Title: "", URL: "https://example.com/empty"},
		{Title: "Producto válido", URL: "https://example.com/valid"},
	}

	// Act
	saved, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), nil, raw)

	// Assert — el producto con título vacío se ignora silenciosamente
	require.NoError(t, err)
	assert.Equal(t, 1, saved, "solo debe guardar el producto con título válido")
}

// --- T-PRD-A03: GetPriceHistory ordered ---

func TestGetPriceHistoryUseCase_ReturnsHistory(t *testing.T) {
	repo := newMockProductRepo()
	tenantID := uuid.New()
	sourceID := uuid.New()

	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID: tenantID, SourceID: sourceID,
		Title: "Prod", URL: "https://x.com/prod",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Prod", "https://x.com/prod"),
	})
	_, _ = repo.Upsert(context.Background(), p)
	_ = repo.SavePriceRecord(context.Background(), value_object.NewPriceRecord(tenantID, p.ID, 50.0))
	_ = repo.SavePriceRecord(context.Background(), value_object.NewPriceRecord(tenantID, p.ID, 60.0))

	uc := usecase.NewGetPriceHistoryUseCase(repo)
	history, err := uc.Execute(context.Background(), tenantID, p.ID)

	require.NoError(t, err)
	assert.Len(t, history, 2)
}
