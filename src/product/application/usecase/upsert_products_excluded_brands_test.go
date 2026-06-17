package usecase_test

// T-PRD-B01: Filtro de marcas excluidas en UpsertProductsUseCase
//
// Verifica que los productos cuya marca (brand) esté en excluded_brands
// NO se persistan, mientras que los de marca permitida sí lo hagan.
// El matching es case-insensitive y trim-insensitive.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpsertProductsUseCase_ExcludedBrands_FiltersExact verifica que una marca
// exactamente igual a la lista excluida no se persista.
func TestUpsertProductsUseCase_ExcludedBrands_FiltersExact(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
	tenantID := uuid.New()
	sourceID := uuid.New()

	price := 100.0
	raw := []scrapeport.RawProduct{
		{Title: "Alfajor Molto", Brand: "molto", URL: "https://example.com/molto1", Price: &price},
	}

	created, updated, err := uc.ExecuteDetailedWithFilter(context.Background(), tenantID, sourceID, nil, raw, []string{"molto"})

	require.NoError(t, err)
	assert.Equal(t, 0, created+updated, "producto con marca excluida no debe persistirse")
	assert.Empty(t, repo.products, "el repo debe quedar vacío")
}

// TestUpsertProductsUseCase_ExcludedBrands_CaseInsensitive verifica que el matching
// ignore mayúsculas/minúsculas: "molto" excluye "Molto" y "MOLTO".
func TestUpsertProductsUseCase_ExcludedBrands_CaseInsensitive(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
	tenantID := uuid.New()
	sourceID := uuid.New()

	price := 50.0
	raw := []scrapeport.RawProduct{
		{Title: "Alfajor Molto mayus", Brand: "Molto", URL: "https://example.com/molto2", Price: &price},
		{Title: "Alfajor MOLTO todo mayus", Brand: "MOLTO", URL: "https://example.com/molto3", Price: &price},
	}

	// Excluded brands configurados en minúsculas — igual que vendrían de DB
	created, updated, err := uc.ExecuteDetailedWithFilter(context.Background(), tenantID, sourceID, nil, raw, []string{"molto"})

	require.NoError(t, err)
	assert.Equal(t, 0, created+updated, "marcas en distintas capitalizaciones deben ser filtradas")
	assert.Empty(t, repo.products, "ningún producto debe persistirse")
}

// TestUpsertProductsUseCase_ExcludedBrands_AllowsOtherBrands verifica que productos
// con marcas distintas a las excluidas sí se persistan.
func TestUpsertProductsUseCase_ExcludedBrands_AllowsOtherBrands(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
	tenantID := uuid.New()
	sourceID := uuid.New()

	price := 80.0
	raw := []scrapeport.RawProduct{
		{Title: "Galletitas Oreo", Brand: "Oreo", URL: "https://example.com/oreo", Price: &price},
	}

	created, updated, err := uc.ExecuteDetailedWithFilter(context.Background(), tenantID, sourceID, nil, raw, []string{"molto"})

	require.NoError(t, err)
	assert.Equal(t, 1, created+updated, "marca no excluida debe persistirse")
	assert.Len(t, repo.products, 1)
}

// TestUpsertProductsUseCase_ExcludedBrands_MixedBatch verifica que en un batch mixto
// solo se filtren los productos con marca excluida y se persistan los permitidos.
func TestUpsertProductsUseCase_ExcludedBrands_MixedBatch(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
	tenantID := uuid.New()
	sourceID := uuid.New()

	price := 90.0
	raw := []scrapeport.RawProduct{
		{Title: "Alfajor Molto", Brand: "MOLTO", URL: "https://example.com/molto-mix", Price: &price},
		{Title: "Galletitas Oreo", Brand: "oreo", URL: "https://example.com/oreo-mix", Price: &price},
		{Title: "Mantecol Original", Brand: "Mantecol", URL: "https://example.com/mantecol-mix", Price: &price},
	}

	created, updated, err := uc.ExecuteDetailedWithFilter(
		context.Background(), tenantID, sourceID, nil, raw,
		[]string{"molto", "mantecol"},
	)

	require.NoError(t, err)
	assert.Equal(t, 1, created+updated, "solo Oreo debe persistirse")
	assert.Len(t, repo.products, 1)
	for _, p := range repo.products {
		assert.Equal(t, "Galletitas Oreo", p.Title)
	}
}

// TestUpsertProductsUseCase_ExcludedBrands_EmptyList verifica que sin marcas excluidas
// todos los productos se persistan normalmente (compatibilidad backward).
func TestUpsertProductsUseCase_ExcludedBrands_EmptyList(t *testing.T) {
	repo := newMockProductRepo()
	uc := usecase.NewUpsertProductsUseCase(repo, newMockSourceCategoryFinder())
	tenantID := uuid.New()
	sourceID := uuid.New()

	price := 70.0
	raw := []scrapeport.RawProduct{
		{Title: "Producto A", Brand: "MarcaA", URL: "https://example.com/a", Price: &price},
		{Title: "Producto B", Brand: "MarcaB", URL: "https://example.com/b", Price: &price},
	}

	created, updated, err := uc.ExecuteDetailedWithFilter(context.Background(), tenantID, sourceID, nil, raw, []string{})

	require.NoError(t, err)
	assert.Equal(t, 2, created+updated)
	assert.Len(t, repo.products, 2)
}
