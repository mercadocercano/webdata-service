package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TEST-ID: T-PRD-D01

func ptrFloat(f float64) *float64 { return &f }

func aNewProduct() entity.CreateProductParams {
	tenantID := uuid.New()
	sourceID := uuid.New()
	return entity.CreateProductParams{
		TenantID:    tenantID,
		SourceID:    sourceID,
		Title:       "Arroz Largo Fino La Criolla 1kg",
		Price:       ptrFloat(450.00),
		Currency:    "ARS",
		URL:         "https://example.com/arroz-la-criolla-1kg",
		ContentHash: value_object.GenerateContentHash(tenantID, sourceID, "Arroz Largo Fino La Criolla 1kg", "https://example.com/arroz-la-criolla-1kg"),
	}
}

func TestScrapedProduct_Create_WithValidParams_Succeeds(t *testing.T) {
	params := aNewProduct()

	p, err := entity.NewScrapedProduct(params)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, params.TenantID, p.TenantID)
	assert.Equal(t, params.Title, p.Title)
	assert.Equal(t, *params.Price, *p.Price)
	assert.Equal(t, "ARS", p.Currency)
	assert.NotEmpty(t, p.ContentHash)
}

func TestScrapedProduct_Create_WithEmptyTitle_ReturnsError(t *testing.T) {
	params := aNewProduct()
	params.Title = ""

	_, err := entity.NewScrapedProduct(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestScrapedProduct_Create_WithEmptyURL_Succeeds(t *testing.T) {
	// URL is optional — callers use source.BaseURL as fallback when not available.
	params := aNewProduct()
	params.URL = ""

	p, err := entity.NewScrapedProduct(params)

	assert.NoError(t, err)
	assert.Equal(t, "", p.URL)
}

func TestScrapedProduct_HasPriceChanged_WhenPriceDiffers_ReturnsTrue(t *testing.T) {
	params := aNewProduct()
	p, _ := entity.NewScrapedProduct(params)

	assert.True(t, p.HasPriceChanged(500.00))
}

func TestScrapedProduct_HasPriceChanged_WhenPriceSame_ReturnsFalse(t *testing.T) {
	params := aNewProduct()
	p, _ := entity.NewScrapedProduct(params)

	assert.False(t, p.HasPriceChanged(450.00))
}

func TestScrapedProduct_HasPriceChanged_WhenOriginalNil_ReturnsFalse(t *testing.T) {
	params := aNewProduct()
	params.Price = nil
	p, _ := entity.NewScrapedProduct(params)

	assert.False(t, p.HasPriceChanged(450.00))
}

func TestScrapedProduct_UpdatePrice_SetsPriceChangedAt(t *testing.T) {
	params := aNewProduct()
	p, _ := entity.NewScrapedProduct(params)

	p.UpdatePrice(550.00)

	assert.Equal(t, 550.00, *p.Price)
	assert.NotNil(t, p.PriceChangedAt)
}

// TEST-ID: T-PRD-D02
func TestContentHash_Generate_IsDeterministic(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()
	title := "Coca Cola 1.5L"
	url := "https://example.com/coca-cola-1-5l"

	hash1 := value_object.GenerateContentHash(tenantID, sourceID, title, url)
	hash2 := value_object.GenerateContentHash(tenantID, sourceID, title, url)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA256 hex length
}

func TestContentHash_Generate_DiffersForDifferentInputs(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()

	hash1 := value_object.GenerateContentHash(tenantID, sourceID, "Product A", "https://example.com/a")
	hash2 := value_object.GenerateContentHash(tenantID, sourceID, "Product B", "https://example.com/b")

	assert.NotEqual(t, hash1, hash2)
}
