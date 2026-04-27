package mercadolibre_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/mercadolibre"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMLAdapter(tokenURL, searchURL string) *mercadolibre.MLAdapter {
	return mercadolibre.NewMLAdapterWithURLs(tokenURL, searchURL, "test-client-id", "test-client-secret")
}

func TestMLAdapter_FindByGTIN_ReturnsProductData(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"name":      "Coca-Cola Zero 500ml",
					"domain_id": "MLA-SOFT_DRINKS",
					"pictures":  []map[string]string{{"url": "https://ml.static.com/img.jpg"}},
					"attributes": []map[string]interface{}{
						{"id": "BRAND", "values": []map[string]string{{"name": "Coca-Cola"}}},
						{"id": "EAN", "values": []map[string]string{{"name": "7790895000430"}}},
					},
				},
			},
		})
	}))
	defer searchServer.Close()

	adapter := newTestMLAdapter(tokenServer.URL+"/oauth/token", searchServer.URL+"/products/search")

	data, err := adapter.FindByGTIN(context.Background(), "7790895000430")

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, "Coca-Cola Zero 500ml", data.Name)
	assert.Equal(t, "Coca-Cola", data.Brand)
	assert.Equal(t, "SOFT_DRINKS", data.Category)
	assert.Equal(t, "https://ml.static.com/img.jpg", data.ImageURL)
	assert.Equal(t, "7790895000430", data.EAN)
	assert.Equal(t, entity.SourceML, data.Source)
}

func TestMLAdapter_FindByGTIN_EmptyResults_ReturnsNil(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
		})
	}))
	defer searchServer.Close()

	adapter := newTestMLAdapter(tokenServer.URL+"/oauth/token", searchServer.URL+"/products/search")

	data, err := adapter.FindByGTIN(context.Background(), "0000000000000")

	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestMLAdapter_MissingCredentials_ReturnsNil(t *testing.T) {
	adapter := mercadolibre.NewMLAdapterWithURLs("", "", "", "")

	data, err := adapter.FindByGTIN(context.Background(), "7790895000430")

	require.NoError(t, err)
	assert.Nil(t, data)
}
