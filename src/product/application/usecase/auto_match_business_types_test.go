package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock BusinessTypeProvider ---

type mockBusinessTypeProvider struct {
	types []port.BusinessType
	err   error
}

func (m *mockBusinessTypeProvider) FetchAll(_ context.Context, _ string) ([]port.BusinessType, error) {
	return m.types, m.err
}

func defaultBusinessTypes() []port.BusinessType {
	return []port.BusinessType{
		{Code: "almacen", Name: "Almacén"},
		{Code: "supermercado", Name: "Supermercado"},
		{Code: "autoservicio", Name: "Autoservicio"},
		{Code: "kiosco", Name: "Kiosco"},
		{Code: "carniceria", Name: "Carnicería"},
		{Code: "verduleria", Name: "Verdulería"},
		{Code: "panaderia", Name: "Panadería"},
		{Code: "farmacia", Name: "Farmacia"},
		{Code: "perfumeria", Name: "Perfumería"},
		{Code: "vinoteca", Name: "Vinoteca"},
		{Code: "fiambreria", Name: "Fiambrería"},
		{Code: "pescaderia", Name: "Pescadería"},
		{Code: "veterinaria", Name: "Veterinaria"},
		{Code: "pet_shop", Name: "Pet Shop"},
		{Code: "dietetica", Name: "Dietética"},
		{Code: "fruteria", Name: "Frutería"},
		{Code: "granja", Name: "Granja"},
	}
}

func createTestProduct(tenantID, sourceID uuid.UUID, title, category, normalizedCat string) *entity.ScrapedProduct {
	p, _ := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID:           tenantID,
		SourceID:           sourceID,
		Title:              title,
		URL:                "https://example.com/" + uuid.New().String(),
		Category:           category,
		NormalizedCategory: normalizedCat,
		ContentHash:        value_object.GenerateContentHash(tenantID, sourceID, title, uuid.New().String()),
	})
	return p
}

// T-007: Auto-match genera propuestas por categoria con marcas populares
func TestAutoMatchBusinessTypes_GeneratesProposalsByCategory(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	// Create products in various categories
	aceite := createTestProduct(tenantID, sourceID, "Aceite Cocinero 1.5L", "Aceites y Vinagres", "aceites")
	carne := createTestProduct(tenantID, sourceID, "Nalga 1kg", "Carnes Vacunas", "carnes")
	verdura := createTestProduct(tenantID, sourceID, "Tomate Redondo", "Verduras", "verduras")

	for _, p := range []*entity.ScrapedProduct{aceite, carne, verdura} {
		_, _ = repo.Upsert(context.Background(), p)
	}

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalProducts)
	assert.Equal(t, 3, result.MatchedCount)
	assert.Equal(t, 0, result.UnmatchedCount)
	assert.Len(t, result.Proposals, 3)

	// Verify each product got matched to correct business types
	proposalMap := make(map[uuid.UUID]usecase.AutoMatchProposal)
	for _, p := range result.Proposals {
		proposalMap[p.ProductID] = p
	}

	// Aceites → almacen (único candidato válido en el catálogo)
	aceiteProposal := proposalMap[aceite.ID]
	assert.NotEmpty(t, aceiteProposal.ProposedBusinessTypes)
	assert.Equal(t, "almacen", aceiteProposal.ProposedBusinessTypes[0].Code)
	assert.Contains(t, aceiteProposal.ProposedBusinessTypes[0].MatchReason, "category_match")
	assert.GreaterOrEqual(t, aceiteProposal.ConfidenceScore, 0.7)

	// Carnes → carniceria (primer candidato) + almacen como alternativa.
	carneProposal := proposalMap[carne.ID]
	assert.NotEmpty(t, carneProposal.ProposedBusinessTypes)
	assert.Equal(t, "carniceria", carneProposal.ProposedBusinessTypes[0].Code)

	// Verduras → verduleria (primer candidato) + almacen como alternativa.
	verduraProposal := proposalMap[verdura.ID]
	assert.NotEmpty(t, verduraProposal.ProposedBusinessTypes)
	assert.Equal(t, "verduleria", verduraProposal.ProposedBusinessTypes[0].Code)
}

// T-007 extension: max 3 business type options per product
func TestAutoMatchBusinessTypes_Max3OptionsPerProduct(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	p := createTestProduct(tenantID, sourceID, "Aceite Girasol 900ml", "Aceites", "aceites")
	_, _ = repo.Upsert(context.Background(), p)

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	require.Len(t, result.Proposals, 1)
	assert.LessOrEqual(t, len(result.Proposals[0].ProposedBusinessTypes), 3)
}

// TestAutoMatchBusinessTypes_NuevosRubrosCorrectos verifica que los rubros nuevos
// (vinoteca, carniceria, verduleria, veterinaria, panaderia) aparezcan como
// primer candidato en sus categorías respectivas.
func TestAutoMatchBusinessTypes_NuevosRubrosCorrectos(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	vino := createTestProduct(tenantID, sourceID, "Vino Malbec 750ml", "Vinos Tintos", "vinos")
	carne := createTestProduct(tenantID, sourceID, "Nalga 1kg", "Carnes Vacunas", "carnes")
	verdura := createTestProduct(tenantID, sourceID, "Tomate Redondo", "Verduras", "verduras")
	mascota := createTestProduct(tenantID, sourceID, "Alimento perro adulto", "Mascotas", "mascotas")
	pan := createTestProduct(tenantID, sourceID, "Pan de campo 1kg", "Panadería", "panaderia")

	for _, p := range []*entity.ScrapedProduct{vino, carne, verdura, mascota, pan} {
		_, _ = repo.Upsert(context.Background(), p)
	}

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	assert.Equal(t, 5, result.MatchedCount)

	proposalMap := make(map[uuid.UUID]usecase.AutoMatchProposal)
	for _, p := range result.Proposals {
		proposalMap[p.ProductID] = p
	}

	// vinos → vinoteca como primer candidato
	vinoProposal := proposalMap[vino.ID]
	require.NotEmpty(t, vinoProposal.ProposedBusinessTypes)
	assert.Equal(t, "vinoteca", vinoProposal.ProposedBusinessTypes[0].Code, "vinos debe proponer vinoteca primero")

	// carnes → carniceria como primer candidato
	carneProposal := proposalMap[carne.ID]
	require.NotEmpty(t, carneProposal.ProposedBusinessTypes)
	assert.Equal(t, "carniceria", carneProposal.ProposedBusinessTypes[0].Code, "carnes debe proponer carniceria primero")

	// verduras → verduleria como primer candidato
	verduraProposal := proposalMap[verdura.ID]
	require.NotEmpty(t, verduraProposal.ProposedBusinessTypes)
	assert.Equal(t, "verduleria", verduraProposal.ProposedBusinessTypes[0].Code, "verduras debe proponer verduleria primero")

	// mascotas → veterinaria como primer candidato
	mascotaProposal := proposalMap[mascota.ID]
	require.NotEmpty(t, mascotaProposal.ProposedBusinessTypes)
	assert.Equal(t, "veterinaria", mascotaProposal.ProposedBusinessTypes[0].Code, "mascotas debe proponer veterinaria primero")

	// panaderia → panaderia como primer candidato
	panProposal := proposalMap[pan.ID]
	require.NotEmpty(t, panProposal.ProposedBusinessTypes)
	assert.Equal(t, "panaderia", panProposal.ProposedBusinessTypes[0].Code, "panaderia debe proponer panaderia primero")
}

// T-009: Auto-match no aplica sin confirmacion del admin (endpoint solo retorna preview, no persiste)
func TestAutoMatchBusinessTypes_DoesNotPersist(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	p := createTestProduct(tenantID, sourceID, "Leche La Serenisima 1L", "Lácteos", "lacteos")
	_, _ = repo.Upsert(context.Background(), p)

	// Verify product has no business types before
	assert.Empty(t, repo.products[p.ID.String()].BusinessTypes)

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.MatchedCount)

	// Verify product STILL has no business types after — use case is preview-only
	assert.Empty(t, repo.products[p.ID.String()].BusinessTypes,
		"auto-match must NOT persist assignments — it is preview-only")
}

func TestAutoMatchBusinessTypes_SkipsAlreadyAssigned(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	// Product with existing assignment
	p := createTestProduct(tenantID, sourceID, "Arroz Gallo 1kg", "Arroz", "arroz")
	_, _ = repo.Upsert(context.Background(), p)
	bt, _ := value_object.NewBusinessTypeAssignment("almacen", "Almacén")
	repo.products[p.ID.String()].BusinessTypes = []value_object.BusinessTypeAssignment{bt}

	// Product without assignment
	p2 := createTestProduct(tenantID, sourceID, "Fideos Matarazzo 500g", "Fideos", "fideos")
	_, _ = repo.Upsert(context.Background(), p2)

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{
		IncludeAlreadyAssigned: false,
	})

	require.NoError(t, err)
	// Only the unassigned product should be matched
	assert.Equal(t, 1, result.TotalProducts)
	assert.Equal(t, 1, result.MatchedCount)
}

func TestAutoMatchBusinessTypes_IncludesAlreadyAssigned(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	p := createTestProduct(tenantID, sourceID, "Arroz Gallo 1kg", "Arroz", "arroz")
	_, _ = repo.Upsert(context.Background(), p)
	bt, _ := value_object.NewBusinessTypeAssignment("almacen", "Almacén")
	repo.products[p.ID.String()].BusinessTypes = []value_object.BusinessTypeAssignment{bt}

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{
		IncludeAlreadyAssigned: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalProducts)
	assert.Equal(t, 1, result.MatchedCount)
}

func TestAutoMatchBusinessTypes_FilterByProductIDs(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	p1 := createTestProduct(tenantID, sourceID, "Aceite 1", "Aceites", "aceites")
	p2 := createTestProduct(tenantID, sourceID, "Aceite 2", "Aceites", "aceites")
	p3 := createTestProduct(tenantID, sourceID, "Aceite 3", "Aceites", "aceites")
	for _, p := range []*entity.ScrapedProduct{p1, p2, p3} {
		_, _ = repo.Upsert(context.Background(), p)
	}

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{
		ProductIDs: []uuid.UUID{p1.ID, p3.ID},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalProducts)
	assert.Equal(t, 2, result.MatchedCount)
}

func TestAutoMatchBusinessTypes_UnmatchedProductsWithNoCategory(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	// Product with no category
	p := createTestProduct(tenantID, sourceID, "Producto Desconocido", "", "")
	_, _ = repo.Upsert(context.Background(), p)

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalProducts)
	assert.Equal(t, 0, result.MatchedCount)
	assert.Equal(t, 1, result.UnmatchedCount)
}

func TestAutoMatchBusinessTypes_FuzzyMatchCategory(t *testing.T) {
	repo := newMockProductRepo()
	btProvider := &mockBusinessTypeProvider{types: defaultBusinessTypes()}
	tenantID := uuid.New()
	sourceID := uuid.New()

	// NormalizedCategory "vinos" → vinoteca (primer candidato) + almacen como alternativa.
	p := createTestProduct(tenantID, sourceID, "Vino Tinto Malbec", "Vinos Tintos", "vinos")
	_, _ = repo.Upsert(context.Background(), p)

	uc := usecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	result, err := uc.Execute(context.Background(), tenantID, usecase.AutoMatchRequest{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.MatchedCount)
	assert.Equal(t, "vinoteca", result.Proposals[0].ProposedBusinessTypes[0].Code)
}
