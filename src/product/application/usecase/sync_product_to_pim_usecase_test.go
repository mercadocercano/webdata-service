package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock PIM Catalog Syncer ---

type mockPIMSyncer struct {
	eanProducts       map[string]*port.PIMGlobalProduct
	nameBrandProducts map[string]*port.PIMGlobalProduct
	created           []port.CreatePIMProductRequest
	updated           []updateCall
	refreshCalled     bool
	createErr         error
	updateErr         error
	searchErr         error
	// spy: trackea qué método de búsqueda se invocó y con qué argumento.
	searchByEANCalls       []string
	searchByNameBrandCalls []string
}

type updateCall struct {
	ID  uuid.UUID
	Req port.UpdatePIMProductRequest
}

func newMockPIMSyncer() *mockPIMSyncer {
	return &mockPIMSyncer{
		eanProducts:       make(map[string]*port.PIMGlobalProduct),
		nameBrandProducts: make(map[string]*port.PIMGlobalProduct),
	}
}

func (m *mockPIMSyncer) SearchByEAN(_ context.Context, _ uuid.UUID, ean string) (*port.PIMGlobalProduct, error) {
	m.searchByEANCalls = append(m.searchByEANCalls, ean)
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	p, ok := m.eanProducts[ean]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockPIMSyncer) SearchByNameBrand(_ context.Context, _ uuid.UUID, name, brand string) (*port.PIMGlobalProduct, error) {
	m.searchByNameBrandCalls = append(m.searchByNameBrandCalls, name+"|"+brand)
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	key := name + "|" + brand
	p, ok := m.nameBrandProducts[key]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockPIMSyncer) Create(_ context.Context, _ uuid.UUID, req port.CreatePIMProductRequest) (*port.PIMGlobalProduct, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = append(m.created, req)
	return &port.PIMGlobalProduct{ID: uuid.New(), Name: req.Name}, nil
}

func (m *mockPIMSyncer) Update(_ context.Context, _ uuid.UUID, id uuid.UUID, req port.UpdatePIMProductRequest) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, updateCall{ID: id, Req: req})
	return nil
}

func (m *mockPIMSyncer) RefreshTemplateProducts(_ context.Context) error {
	m.refreshCalled = true
	return nil
}

func (m *mockPIMSyncer) ListNeedingEnrichment(_ context.Context, _ string, _, _ int) (*port.ListNeedingEnrichmentResult, error) {
	return &port.ListNeedingEnrichmentResult{}, nil
}

// --- Helper builders ---

func buildSyncableProduct() *entity.ScrapedProduct {
	price := 450.0
	tenantID := uuid.New()
	sourceID := uuid.New()
	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID:    tenantID,
		SourceID:    sourceID,
		Title:       "Coca Cola 2.25L",
		Price:       &price,
		Currency:    "ARS",
		URL:         "https://example.com/coca-cola",
		ImageURL:    "https://example.com/coca.jpg",
		Brand:       "Coca Cola",
		Category:    "Bebidas",
		EAN:         "7790895000119",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Coca Cola 2.25L", "https://example.com/coca-cola"),
	})
	bt, _ := value_object.NewBusinessTypeAssignment("almacen", "Almacén de Barrio")
	p.AssignBusinessTypes([]value_object.BusinessTypeAssignment{bt})
	return p
}

func buildNotReadyProduct() *entity.ScrapedProduct {
	tenantID := uuid.New()
	sourceID := uuid.New()
	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID:    tenantID,
		SourceID:    sourceID,
		Title:       "Producto incompleto",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Producto incompleto", ""),
	})
	return p
}

// --- Tests: Execute ---

func TestSyncProductToPIM_WhenNotReady_SkipsSilently(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildNotReadyProduct()

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	assert.Empty(t, syncer.created)
	assert.Empty(t, syncer.updated)
}

func TestSyncProductToPIM_WhenAlreadySynced_SkipsSilently(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	product.MarkSyncedToPIM()

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	assert.Empty(t, syncer.created)
	assert.Empty(t, syncer.updated)
}

func TestSyncProductToPIM_NewProduct_CreatesInPIM(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "mitienda")

	require.NoError(t, err)
	require.Len(t, syncer.created, 1)
	assert.Equal(t, "Coca Cola 2.25L", syncer.created[0].Name)
	assert.Equal(t, "Coca Cola", syncer.created[0].Brand)
	assert.Equal(t, "Bebidas", syncer.created[0].Category)
	assert.Equal(t, "7790895000119", syncer.created[0].EAN)
	assert.Equal(t, "scraper_mitienda", syncer.created[0].Source)
	assert.Equal(t, "almacen", syncer.created[0].BusinessType)
	assert.NotNil(t, product.SyncedToPIMAt)
}

func TestSyncProductToPIM_NewProduct_WithoutSourceName_UsesDefaultSource(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "")

	require.NoError(t, err)
	require.Len(t, syncer.created, 1)
	assert.Equal(t, "scraper", syncer.created[0].Source)
}

func TestSyncProductToPIM_DedupByEAN_UpdatesExisting(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	existingID := uuid.New()
	syncer.eanProducts["7790895000119"] = &port.PIMGlobalProduct{
		ID:   existingID,
		Name: "Coca Cola 2.25L",
		EAN:  "7790895000119",
	}
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	assert.Empty(t, syncer.created, "should NOT create a new product")
	require.Len(t, syncer.updated, 1)
	assert.Equal(t, existingID, syncer.updated[0].ID)
	assert.Equal(t, *product.Price, *syncer.updated[0].Req.Price)
	assert.Equal(t, product.ImageURL, syncer.updated[0].Req.ImageURL)
	assert.NotNil(t, product.SyncedToPIMAt)
}

// E25 / ADR-006: el re-sync (updateExisting) propaga el business_type resuelto como
// candidate, igual que createNew. webdata no decide la política §8 — solo aporta el candidato.
func TestSyncProductToPIM_DedupByEAN_PropagatesBusinessTypeCandidate(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	existingID := uuid.New()
	syncer.eanProducts["7790895000119"] = &port.PIMGlobalProduct{
		ID:   existingID,
		Name: "Coca Cola 2.25L",
		EAN:  "7790895000119",
	}
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct() // business_type asignado = "almacen"
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	require.Len(t, syncer.updated, 1)
	assert.Equal(t, "almacen", syncer.updated[0].Req.BusinessType,
		"updateExisting debe propagar el business_type como candidate")
}

func TestSyncProductToPIM_DedupByNameBrand_UpdatesExisting(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	existingID := uuid.New()
	syncer.nameBrandProducts["Coca Cola 2.25L|Coca Cola"] = &port.PIMGlobalProduct{
		ID:    existingID,
		Name:  "Coca Cola 2.25L",
		Brand: "Coca Cola",
	}
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	product.EAN = "" // sin EAN, fallback a nombre+marca
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	assert.Empty(t, syncer.created)
	require.Len(t, syncer.updated, 1)
	assert.Equal(t, existingID, syncer.updated[0].ID)
}

func TestSyncProductToPIM_SearchError_ReturnsError(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	syncer.searchErr = errors.New("PIM unavailable")
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()

	err := uc.Execute(context.Background(), product, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "searching PIM")
	assert.Nil(t, product.SyncedToPIMAt)
}

func TestSyncProductToPIM_CreateError_ReturnsError(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	syncer.createErr = errors.New("PIM create failed")
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()

	err := uc.Execute(context.Background(), product, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating PIM product")
	assert.Nil(t, product.SyncedToPIMAt)
}

func TestSyncProductToPIM_UpdateError_ReturnsError(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	syncer.eanProducts["7790895000119"] = &port.PIMGlobalProduct{ID: uuid.New()}
	syncer.updateErr = errors.New("PIM update failed")
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()

	err := uc.Execute(context.Background(), product, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "updating PIM product")
}

// --- Tests: SyncBatch ---

func TestSyncBatch_MixedProducts_ReturnsCorrectCounts(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	syncable1 := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), syncable1)

	syncable2 := buildSyncableProduct()
	syncable2.Title = "Fanta 2.25L"
	_, _ = repo.Upsert(context.Background(), syncable2)

	notReady := buildNotReadyProduct()

	alreadySynced := buildSyncableProduct()
	alreadySynced.MarkSyncedToPIM()

	products := []*entity.ScrapedProduct{syncable1, syncable2, notReady, alreadySynced}

	synced, skipped, failed := uc.SyncBatch(context.Background(), products, "test")

	assert.Equal(t, 2, synced)
	assert.Equal(t, 2, skipped) // notReady + alreadySynced
	assert.Equal(t, 0, failed)
}

func TestSyncBatch_WithCreateErrors_CountsFailed(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	syncer.createErr = errors.New("PIM down")
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), product)

	synced, skipped, failed := uc.SyncBatch(context.Background(), []*entity.ScrapedProduct{product}, "test")

	assert.Equal(t, 0, synced)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 1, failed)
}

// --- Tests: SyncPending ---

func TestSyncPending_SyncsAllPendingProducts(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)
	tenantID := uuid.New()

	// Crear producto syncable con el tenantID correcto
	price := 100.0
	sourceID := uuid.New()
	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID:    tenantID,
		SourceID:    sourceID,
		Title:       "Producto Real",
		Price:       &price,
		ImageURL:    "https://img.com/p.jpg",
		Category:    "General",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Producto Real", ""),
	})
	_, _ = repo.Upsert(context.Background(), p)

	synced, failed, err := uc.SyncPending(context.Background(), tenantID, "test")

	require.NoError(t, err)
	assert.Equal(t, 1, synced)
	assert.Equal(t, 0, failed)
	assert.True(t, syncer.refreshCalled, "should refresh template products after sync")
}

func TestSyncPending_NoProducts_SkipsRefresh(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	synced, failed, err := uc.SyncPending(context.Background(), uuid.New(), "test")

	require.NoError(t, err)
	assert.Equal(t, 0, synced)
	assert.Equal(t, 0, failed)
	assert.False(t, syncer.refreshCalled, "should NOT refresh when nothing was synced")
}

// --- Tests: Product without business types ---

func TestSyncProductToPIM_WithoutBusinessTypes_SendsEmptyBusinessType(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	product.BusinessTypes = nil
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	require.Len(t, syncer.created, 1)
	assert.Empty(t, syncer.created[0].BusinessType)
}

// --- Tests: EAN priority over name+brand ---

func TestSyncProductToPIM_EANMatchTakesPriority_OverNameBrand(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()

	eanID := uuid.New()
	nameBrandID := uuid.New()
	syncer.eanProducts["7790895000119"] = &port.PIMGlobalProduct{ID: eanID}
	syncer.nameBrandProducts["Coca Cola 2.25L|Coca Cola"] = &port.PIMGlobalProduct{ID: nameBrandID}

	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)
	product := buildSyncableProduct()
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err)
	require.Len(t, syncer.updated, 1)
	assert.Equal(t, eanID, syncer.updated[0].ID, "should match by EAN, not name+brand")
}

// --- Tests: markSynced tolerates repo errors ---

func TestSyncProductToPIM_MarkSyncedFails_DoesNotReturnError(t *testing.T) {
	// Use a repo where MarkSyncedToPIM fails
	repo := &markSyncFailRepo{}
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()

	err := uc.Execute(context.Background(), product, "test")

	require.NoError(t, err, "markSynced failure should not propagate")
	assert.NotNil(t, product.SyncedToPIMAt, "in-memory mark should still happen")
	require.Len(t, syncer.created, 1)
}

// markSyncFailRepo: repo that fails only on MarkSyncedToPIM
type markSyncFailRepo struct {
	mockProductRepo
}

func (r *markSyncFailRepo) MarkSyncedToPIM(_ context.Context, _, _ uuid.UUID) error {
	return errors.New("db connection lost")
}

// --- Tests E19: bypass de EAN inválido en sync webdata→PIM ---
//
// Regresión crítica: un EAN inválido (corto, o 13 dígitos con checksum incorrecto)
// NO debe descartar el producto. Debe sincronizar con EAN="" y deduplicar por nombre+marca.
// Si este comportamiento se rompe, ~600 productos se pierden silenciosamente.

// T-E19-01: EAN inválido (caso exacto del log: 7796074860226, 13 dígitos, checksum incorrecto)
// → se crea en PIM con req.EAN="" y se busca por nombre+marca, NO por EAN.
func TestSyncProductToPIM_InvalidEAN_13Digits_BadChecksum_SyncsWithEmptyEAN(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	product.EAN = "7796074860226" // caso exacto del log E19: 13 dígitos, checksum inválido
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "caprabo")

	require.NoError(t, err, "EAN inválido NO debe descartar el producto ni retornar error")

	// El producto debe haberse creado (no actualizado — no hay match previo)
	require.Len(t, syncer.created, 1, "debe crear el producto en PIM")
	assert.Equal(t, "", syncer.created[0].EAN,
		"EAN inválido debe llegar al PIM como string vacío, nunca como el código corrupto")
	assert.Equal(t, "Coca Cola 2.25L", syncer.created[0].Name, "el nombre debe preservarse")

	// La búsqueda NO debe haber pasado por SearchByEAN
	assert.Empty(t, syncer.searchByEANCalls,
		"con EAN inválido NO se debe llamar SearchByEAN — causaría HTTP 400 en PIM")

	// La búsqueda SÍ debe haber pasado por SearchByNameBrand (dedup fallback)
	require.Len(t, syncer.searchByNameBrandCalls, 1,
		"con EAN inválido el dedup debe usar nombre+marca")

	assert.NotNil(t, product.SyncedToPIMAt, "el producto debe quedar marcado como sincronizado")
}

// T-E19-02: EAN código interno VTEX corto ("210165") → mismo comportamiento que T-E19-01.
func TestSyncProductToPIM_InvalidEAN_ShortCode_SyncsWithEmptyEAN(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()
	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)

	product := buildSyncableProduct()
	product.EAN = "210165" // código interno VTEX — 6 dígitos, no es EAN-13
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "vtex")

	require.NoError(t, err, "código interno corto NO debe descartar el producto")
	require.Len(t, syncer.created, 1)
	assert.Equal(t, "", syncer.created[0].EAN,
		"código PLU corto debe normalizarse a vacío antes de llegar al PIM")
	assert.Empty(t, syncer.searchByEANCalls,
		"código PLU corto no debe disparar SearchByEAN")
	require.Len(t, syncer.searchByNameBrandCalls, 1)
}

// T-E19-03: EAN inválido + existe match por nombre+marca en PIM
// → va por updateExisting (no crea duplicado).
// Este es el caso donde el producto ya estaba en PIM desde una sincronización anterior válida.
func TestSyncProductToPIM_InvalidEAN_MatchByNameBrand_UpdatesExistingNoDuplicate(t *testing.T) {
	repo := newMockProductRepo()
	syncer := newMockPIMSyncer()

	existingID := uuid.New()
	// El producto ya existe en PIM indexado por nombre+marca (sincronizado previamente con EAN válido
	// o sin EAN). Ahora llega con EAN corrupto — debe actualizar, no crear un duplicado.
	syncer.nameBrandProducts["Coca Cola 2.25L|Coca Cola"] = &port.PIMGlobalProduct{
		ID:    existingID,
		Name:  "Coca Cola 2.25L",
		Brand: "Coca Cola",
	}

	uc := usecase.NewSyncProductToPIMUseCase(repo, syncer)
	product := buildSyncableProduct()
	product.EAN = "7796074860226" // EAN corrupto del retailer
	_, _ = repo.Upsert(context.Background(), product)

	err := uc.Execute(context.Background(), product, "caprabo")

	require.NoError(t, err)
	assert.Empty(t, syncer.created, "NO debe crear un duplicado cuando ya existe por nombre+marca")
	require.Len(t, syncer.updated, 1, "debe actualizar el producto existente")
	assert.Equal(t, existingID, syncer.updated[0].ID,
		"debe actualizar el ID correcto del producto existente en PIM")
	assert.Empty(t, syncer.searchByEANCalls,
		"con EAN inválido no se debe haber llamado SearchByEAN")
	require.Len(t, syncer.searchByNameBrandCalls, 1,
		"el dedup fue por nombre+marca")
	assert.NotNil(t, product.SyncedToPIMAt)
}

// Suppress unused import warnings
var _ = time.Now
