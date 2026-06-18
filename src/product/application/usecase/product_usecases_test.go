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

// --- Mock source category finder ---

type mockSourceCategoryFinder struct {
	categories    map[string]string // sourceID -> category
	authoritative map[string]bool   // sourceID -> authoritative_category
}

func newMockSourceCategoryFinder() *mockSourceCategoryFinder {
	return &mockSourceCategoryFinder{
		categories:    make(map[string]string),
		authoritative: make(map[string]bool),
	}
}

func (m *mockSourceCategoryFinder) FindCategoryBySourceID(_ context.Context, _, sourceID uuid.UUID) (string, bool, error) {
	cat, ok := m.categories[sourceID.String()]
	if !ok {
		return "", false, errors.New("source not found")
	}
	return cat, m.authoritative[sourceID.String()], nil
}

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

func (r *failingProductRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *failingProductRepo) BulkSoftDelete(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *failingProductRepo) UpdateBlocked(_ context.Context, _, _ uuid.UUID, _ bool) error {
	return nil
}
func (r *failingProductRepo) SaveBusinessTypes(_ context.Context, _, _ uuid.UUID, _ []value_object.BusinessTypeAssignment) error {
	return nil
}
func (r *failingProductRepo) RemoveBusinessType(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (r *failingProductRepo) FindBusinessTypesForProduct(_ context.Context, _, _ uuid.UUID) ([]value_object.BusinessTypeAssignment, error) {
	return nil, nil
}

func (r *failingProductRepo) GetFilters(_ context.Context, _ uuid.UUID) (*port.ProductFilters, error) {
	return &port.ProductFilters{}, nil
}

func (r *failingProductRepo) MarkSyncedToPIM(_ context.Context, _, _ uuid.UUID) error {
	return nil
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
		if filter.WithoutBusinessTypes && len(p.BusinessTypes) > 0 {
			continue
		}
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

func (m *mockProductRepo) SoftDelete(_ context.Context, _, id uuid.UUID) error {
	if _, ok := m.products[id.String()]; !ok {
		return errors.New("product not found")
	}
	delete(m.products, id.String())
	return nil
}

func (m *mockProductRepo) BulkSoftDelete(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (int64, error) {
	var count int64
	for _, id := range ids {
		if _, ok := m.products[id.String()]; ok {
			delete(m.products, id.String())
			count++
		}
	}
	return count, nil
}

func (m *mockProductRepo) UpdateBlocked(_ context.Context, _, id uuid.UUID, blocked bool) error {
	p, ok := m.products[id.String()]
	if !ok {
		return errors.New("product not found")
	}
	p.IsBlocked = blocked
	return nil
}

func (m *mockProductRepo) SaveBusinessTypes(_ context.Context, _, productID uuid.UUID, assignments []value_object.BusinessTypeAssignment) error {
	p, ok := m.products[productID.String()]
	if !ok {
		return errors.New("product not found")
	}
	p.BusinessTypes = assignments
	return nil
}

func (m *mockProductRepo) RemoveBusinessType(_ context.Context, _, productID uuid.UUID, code string) error {
	p, ok := m.products[productID.String()]
	if !ok {
		return errors.New("product not found")
	}
	var filtered []value_object.BusinessTypeAssignment
	for _, bt := range p.BusinessTypes {
		if bt.BusinessTypeCode != code {
			filtered = append(filtered, bt)
		}
	}
	p.BusinessTypes = filtered
	return nil
}

func (m *mockProductRepo) FindBusinessTypesForProduct(_ context.Context, _, productID uuid.UUID) ([]value_object.BusinessTypeAssignment, error) {
	p, ok := m.products[productID.String()]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p.BusinessTypes, nil
}

func (m *mockProductRepo) GetFilters(_ context.Context, _ uuid.UUID) (*port.ProductFilters, error) {
	return &port.ProductFilters{}, nil
}

func (m *mockProductRepo) MarkSyncedToPIM(_ context.Context, _, _ uuid.UUID) error {
	return nil
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
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
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
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
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
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
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
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
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

// --- T-PRD-A05: DeleteProduct ---

func TestDeleteProductUseCase_SoftDeletes(t *testing.T) {
	repo := newMockProductRepo()
	tenantID := uuid.New()
	sourceID := uuid.New()

	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID: tenantID, SourceID: sourceID,
		Title: "To delete", URL: "https://x.com/del",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "To delete", "https://x.com/del"),
	})
	_, _ = repo.Upsert(context.Background(), p)

	uc := usecase.NewDeleteProductUseCase(repo)
	err := uc.Execute(context.Background(), tenantID, p.ID)
	require.NoError(t, err)
	assert.Len(t, repo.products, 0)
}

func TestDeleteProductUseCase_NotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewDeleteProductUseCase(repo)
	err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
}

// --- T-PRD-A06: BulkDeleteProducts ---

func TestBulkDeleteProductsUseCase_DeletesMultiple(t *testing.T) {
	repo := newMockProductRepo()
	tenantID := uuid.New()
	sourceID := uuid.New()

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
			TenantID: tenantID, SourceID: sourceID,
			Title: "Prod", URL: "https://x.com/" + uuid.New().String(),
			ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Prod", uuid.New().String()),
		})
		_, _ = repo.Upsert(context.Background(), p)
		ids = append(ids, p.ID)
	}

	uc := usecase.NewBulkDeleteProductsUseCase(repo)
	deleted, err := uc.Execute(context.Background(), tenantID, ids[:2])
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	assert.Len(t, repo.products, 1)
}

// --- T-PRD-A07: UpdateProduct (block) ---

func TestUpdateProductUseCase_BlocksProduct(t *testing.T) {
	repo := newMockProductRepo()
	tenantID := uuid.New()
	sourceID := uuid.New()

	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID: tenantID, SourceID: sourceID,
		Title: "To block", URL: "https://x.com/block",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "To block", "https://x.com/block"),
	})
	_, _ = repo.Upsert(context.Background(), p)

	uc := usecase.NewUpdateProductUseCase(repo)
	err := uc.Execute(context.Background(), tenantID, p.ID, true)
	require.NoError(t, err)
	assert.True(t, repo.products[p.ID.String()].IsBlocked)
}

func TestUpdateProductUseCase_NotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpdateProductUseCase(repo)
	err := uc.Execute(context.Background(), uuid.New(), uuid.New(), true)
	assert.Error(t, err)
}

// --- T-PRD-A08: Auto-assign business types on new product ---

func TestUpsertProductsUseCase_AssignsBusinessType_NewProduct(t *testing.T) {
	// Arrange
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "supermercado"

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price},
	}

	// Act
	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, saved)

	// Find the created product and check business types
	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1, "new product should have 1 business type assigned")
		assert.Equal(t, "almacen", p.BusinessTypes[0].BusinessTypeCode)
		assert.Equal(t, "Almacén de Barrio", p.BusinessTypes[0].BusinessTypeName)
	}
}

func TestUpsertProductsUseCase_DoesNotOverwriteExistingBusinessTypes(t *testing.T) {
	// Arrange — product already has a business type
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "supermercado"

	// Create product first
	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Leche 1L", URL: "https://example.com/leche", Price: &price},
	}
	_, _ = uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Verify product has 1 business type
	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1)
	}

	// Act — upsert same product again (update path)
	price2 := 120.0
	raw[0].Price = &price2
	_, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert — business types should NOT be overwritten
	require.NoError(t, err)
	for _, p := range repo.products {
		assert.Len(t, p.BusinessTypes, 1, "existing business types should not be overwritten")
		assert.Equal(t, "almacen", p.BusinessTypes[0].BusinessTypeCode)
	}
}

// E26: fuente AUTORITATIVA (single-rubro) → el mapeo de la fuente manda y se saltea
// el resolver por-producto. Un shampoo de mascota (cuya categoría resolvería a
// perfumeria/peluqueria por keyword) debe quedar en veterinaria por ser Puppis.
func TestUpsertProductsUseCase_AuthoritativeSource_SkipsPerProductResolver(t *testing.T) {
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "puppis_general" // → veterinaria
	finder.authoritative[sourceID.String()] = true

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Shampoo para perros", URL: "https://puppis.com.ar/shampoo", Price: &price, Category: "/Higiene y Belleza/Shampoo/"},
	}

	_, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1)
		assert.Equal(t, "veterinaria", p.BusinessTypes[0].BusinessTypeCode,
			"fuente autoritativa debe usar el mapeo de la fuente, no el resolver por-producto")
	}
}

// Contraste: la MISMA categoría per-producto con fuente NO autoritativa → el resolver
// por-producto gana (shampoo → perfumeria), demostrando que el flag es lo que cambia.
func TestUpsertProductsUseCase_NonAuthoritativeSource_UsesPerProductResolver(t *testing.T) {
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "puppis_general" // → veterinaria (fallback)
	finder.authoritative[sourceID.String()] = false

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Shampoo para perros", URL: "https://puppis.com.ar/shampoo", Price: &price, Category: "/Higiene y Belleza/Shampoo/"},
	}

	_, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1)
		assert.Equal(t, "perfumeria", p.BusinessTypes[0].BusinessTypeCode,
			"fuente no autoritativa debe usar el resolver por-producto (shampoo→perfumeria)")
	}
}

func TestUpsertProductsUseCase_UnknownCategory_SkipsGracefully(t *testing.T) {
	// Arrange
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "unknown_category"

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 50.0
	raw := []scrapeport.RawProduct{
		{Title: "Producto X", URL: "https://example.com/x", Price: &price},
	}

	// Act
	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert — product saved, no business types assigned, no error
	require.NoError(t, err)
	assert.Equal(t, 1, saved)
	for _, p := range repo.products {
		assert.Empty(t, p.BusinessTypes, "unknown category should not assign business types")
	}
}

func TestUpsertProductsUseCase_SourceNotFound_SkipsGracefully(t *testing.T) {
	// Arrange — finder has no category for this source
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 50.0
	raw := []scrapeport.RawProduct{
		{Title: "Producto Y", URL: "https://example.com/y", Price: &price},
	}

	// Act
	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert — product saved, no business types, no error
	require.NoError(t, err)
	assert.Equal(t, 1, saved)
	for _, p := range repo.products {
		assert.Empty(t, p.BusinessTypes, "source not found should not assign business types")
	}
}

// --- T-PRD-E17: Per-product business type resolution ---

// TestUpsertProductsUseCase_PerProductCategory_MixedBatch verifica que un batch con
// categorías distintas por producto reciba business_types distintos, sin que el
// autoAssignment de fuente (supermercado→almacen) sobrescriba al resolver per-producto.
func TestUpsertProductsUseCase_PerProductCategory_MixedBatch(t *testing.T) {
	// Arrange — fuente con category supermercado → almacen por defecto
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "supermercado"

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 50.0
	raw := []scrapeport.RawProduct{
		// Producto con categoría de path estilo Átomo — debe ir a fiambreria
		{
			Title:    "Yogur Natural 180g",
			URL:      "https://example.com/yogur",
			Price:    &price,
			Category: "/Lácteos/Yogures/Yogur en vasos/",
		},
		// Producto con categoría LIMPIEZA en mayúsculas — debe ir a limpieza
		{
			Title:    "Lavandina 2L",
			URL:      "https://example.com/lavandina",
			Price:    &price,
			Category: "LIMPIEZA",
		},
		// Producto con categoría genérica que no mapea per-producto — debe usar fallback (almacen)
		{
			Title:    "Aceite de Oliva 500ml",
			URL:      "https://example.com/aceite",
			Price:    &price,
			Category: "Aceites",
		},
	}

	// Act
	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 3, saved)

	// Recolectamos por título para verificar independientemente del orden de iteración
	byTitle := make(map[string]*entity.ScrapedProduct)
	for _, p := range repo.products {
		byTitle[p.Title] = p
	}

	// Yogur → fiambreria (per-product resolver gana sobre fallback supermercado→almacen)
	yogur := byTitle["Yogur Natural 180g"]
	require.NotNil(t, yogur)
	require.Len(t, yogur.BusinessTypes, 1)
	assert.Equal(t, "fiambreria", yogur.BusinessTypes[0].BusinessTypeCode,
		"yogur debe ser fiambreria, no almacen (fuente supermercado)")

	// Lavandina → limpieza
	lavandina := byTitle["Lavandina 2L"]
	require.NotNil(t, lavandina)
	require.Len(t, lavandina.BusinessTypes, 1)
	assert.Equal(t, "limpieza", lavandina.BusinessTypes[0].BusinessTypeCode)

	// Aceite → almacen (per-product resolver matchea "aceite" → almacen, consistente con fallback)
	aceite := byTitle["Aceite de Oliva 500ml"]
	require.NotNil(t, aceite)
	require.Len(t, aceite.BusinessTypes, 1)
	assert.Equal(t, "almacen", aceite.BusinessTypes[0].BusinessTypeCode)
}

// TestUpsertProductsUseCase_PerProductCategory_FallbackWhenNoMatch verifica que cuando
// la categoría del producto no tiene match en el resolver, se usa el autoAssignment de fuente.
func TestUpsertProductsUseCase_PerProductCategory_FallbackWhenNoMatch(t *testing.T) {
	// Arrange — fuente con category lacteos → fiambreria
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "lacteos"

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 30.0
	raw := []scrapeport.RawProduct{
		// Categoría completamente desconocida — debe caer en el autoAssignment de fuente (fiambreria)
		{
			Title:    "Producto Xyzzy Sin Categoría Conocida",
			URL:      "https://example.com/xyzzy",
			Price:    &price,
			Category: "CategoríaDesconocida99",
		},
	}

	// Act
	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, saved)

	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1, "debe tener el business type del fallback de fuente")
		assert.Equal(t, "fiambreria", p.BusinessTypes[0].BusinessTypeCode,
			"fallback de fuente lacteos→fiambreria debe aplicarse cuando el resolver no matchea")
	}
}

// TestUpsertProductsUseCase_PerProductCategory_LacteosSingleProduct verifica el caso
// puntual de E17: producto Átomo con category lácteos va a fiambreria, NO a almacen.
func TestUpsertProductsUseCase_PerProductCategory_LacteosSingleProduct(t *testing.T) {
	// Fuente con supermercado (que da almacen) para exagerar el contraste
	repo := newMockProductRepo()
	finder := newMockSourceCategoryFinder()
	sourceID := uuid.New()
	tenantID := uuid.New()
	finder.categories[sourceID.String()] = "supermercado"

	uc := usecase.NewUpsertProductsUseCase(repo, finder)
	price := 80.0
	raw := []scrapeport.RawProduct{
		{
			Title:    "Leche La Serenísima Entera 1L",
			URL:      "https://example.com/leche",
			Price:    &price,
			Category: "Leches Larga Vida",
		},
	}

	saved, err := uc.Execute(context.Background(), tenantID, sourceID, nil, raw)
	require.NoError(t, err)
	assert.Equal(t, 1, saved)

	for _, p := range repo.products {
		require.Len(t, p.BusinessTypes, 1)
		assert.Equal(t, "fiambreria", p.BusinessTypes[0].BusinessTypeCode,
			"leche debe ir a fiambreria aunque la fuente sea supermercado")
	}
}

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
