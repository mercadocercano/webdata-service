package openfoodfacts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/openfoodfacts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOFFAdapter_FindByGTIN_ReturnsProductData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "MercadoCercano/1.0 (contacto@mercadocercano.com)", r.Header.Get("User-Agent"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": 1,
			"product": map[string]interface{}{
				"product_name":    "Coca-Cola Zero",
				"brands":          "Coca-Cola",
				"categories_tags": []string{"en:beverages", "en:sodas"},
				"image_front_url": "https://off.images.org/img.jpg",
				"code":            "7790895000430",
			},
		})
	}))
	defer server.Close()

	adapter := openfoodfacts.NewOFFAdapterWithBaseURL(server.URL)

	data, err := adapter.FindByGTIN(context.Background(), "7790895000430")

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, "Coca-Cola Zero", data.Name)
	assert.Equal(t, "Coca-Cola", data.Brand)
	assert.Equal(t, "beverages", data.Category)
	assert.Equal(t, "https://off.images.org/img.jpg", data.ImageURL)
	assert.Equal(t, "7790895000430", data.EAN)
	assert.Equal(t, entity.SourceOFF, data.Source)
}

func TestOFFAdapter_FindByGTIN_Status0_ReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        0,
			"status_verbose": "product not found",
		})
	}))
	defer server.Close()

	adapter := openfoodfacts.NewOFFAdapterWithBaseURL(server.URL)

	data, err := adapter.FindByGTIN(context.Background(), "0000000000000")

	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestOFFAdapter_FindByName_ReturnsProductData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 1,
			"products": []map[string]interface{}{
				{
					"product_name":    "Granola Bar",
					"brands":          "Kelloggs",
					"categories_tags": []string{"en:snacks"},
					"image_front_url": "https://off.images.org/granola.jpg",
					"code":            "12345678",
				},
			},
		})
	}))
	defer server.Close()

	adapter := openfoodfacts.NewOFFAdapterWithBaseURL(server.URL)

	data, err := adapter.FindByName(context.Background(), "Granola Bar")

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, "Granola Bar", data.Name)
	assert.Equal(t, entity.SourceOFF, data.Source)
}
