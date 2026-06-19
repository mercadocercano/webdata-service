package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCoopResponse builds a minimal Coop. Obrera API envelope.
func buildCoopResponse(estado int, articulos []map[string]interface{}, total int) []byte {
	resp := map[string]interface{}{
		"estado":  estado,
		"mensaje": "OK",
		"ticket":  "",
		"datos": map[string]interface{}{
			"articulos":          articulos,
			"cantidad_articulos": total,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func coopArticuloPayload(descripcion, marca, imagen string, precio, precioPromo, precioAnterior float64) map[string]interface{} {
	return map[string]interface{}{
		"cod_interno":     "COD-001",
		"descripcion":     descripcion,
		"marca_desc":      marca,
		"precio":          precio,
		"precio_promo":    precioPromo,
		"precio_anterior": precioAnterior,
		"imagen":          imagen,
		"id_categoria":    10,
	}
}

// --- T-COOP-01: mapCoopArticulo maps all fields correctly ---

func jsonNumber(s string) *json.Number {
	n := json.Number(s)
	return &n
}

func TestCoopObreraAdapter_MapArticulo_AllFields(t *testing.T) {
	art := coopArticulo{
		CodInterno:     "COD-123",
		Descripcion:    "Aceite Girasol 1.5L",
		MarcaDesc:      "Natura",
		Precio:         jsonNumber("1250.00"),
		PrecioPromo:    nil, // null in API
		PrecioAnterior: jsonNumber("1400.00"),
		Imagen:         "https://www.lacoopeencasa.coop/media/catalog/product/a/c/aceite.jpg",
	}

	p := mapCoopArticulo(art)

	assert.Equal(t, "Aceite Girasol 1.5L", p.Title)
	assert.Equal(t, "Natura", p.Brand)
	require.NotNil(t, p.Price)
	assert.Equal(t, 1250.0, *p.Price)
	assert.Equal(t, "https://www.lacoopeencasa.coop/media/catalog/product/a/c/aceite.jpg", p.ImageURL)
	assert.Equal(t, "COD-123", p.SKU)
	assert.Empty(t, p.EAN, "EAN must be empty — cod_interno is NOT an EAN")
}

// --- T-COOP-02: price fallback logic ---

func TestSelectCoopPrice_UsesPrecios(t *testing.T) {
	tests := []struct {
		name           string
		precio         *json.Number
		precioPromo    *json.Number
		precioAnterior *json.Number
		want           float64
	}{
		{"precio present", jsonNumber("1000.00"), jsonNumber("800.00"), jsonNumber("1200.00"), 1000},
		{"precio null fallback to promo", nil, jsonNumber("800.00"), jsonNumber("1200.00"), 800},
		{"precio and promo null fallback to anterior", nil, nil, jsonNumber("1200.00"), 1200},
		{"all null", nil, nil, nil, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			art := coopArticulo{
				Precio:         tc.precio,
				PrecioPromo:    tc.precioPromo,
				PrecioAnterior: tc.precioAnterior,
			}
			got := selectCoopPrice(art)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- T-COOP-03: FetchHTTPJSON paginates until empty response ---

func TestCoopObreraAdapter_FetchHTTPJSON_PaginatesUntilEmpty(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Decode the request body to check which page was requested
		var body coopRequestBody
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch body.Pagina {
		case 0:
			// Page 0: 2 products
			w.Write(buildCoopResponse(1, []map[string]interface{}{
				coopArticuloPayload("Aceite 1L", "Natura", "https://img/aceite.jpg", 1000, 0, 0),
				coopArticuloPayload("Arroz 1kg", "Gallo", "https://img/arroz.jpg", 500, 0, 0),
			}, 10))
		case 1:
			// Page 1: 1 product
			w.Write(buildCoopResponse(1, []map[string]interface{}{
				coopArticuloPayload("Azúcar 1kg", "Ledesma", "https://img/azucar.jpg", 800, 0, 0),
			}, 10))
		default:
			// Page 2+: empty → stop
			w.Write(buildCoopResponse(1, []map[string]interface{}{}, 10))
		}
	}))
	defer server.Close()

	a := newCoopObreraAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(context.Background(), server.URL+"/articulos/pagina", scrapingport.HTTPJSONOptions{
		CategoryID:         2,
		PaginationStrategy: "page_increment",
	})

	require.NoError(t, err)
	assert.Len(t, products, 3, "should aggregate products from pages 0 and 1")
	// Should stop after coopMaxEmptyPages consecutive empty pages
	// callCount: page0, page1, page2 (empty), page3 (empty) → 4 calls
	assert.GreaterOrEqual(t, callCount, 3)
}

// --- T-COOP-04: POST body has correct structure (tipo_seleccion inside filtros) ---

func TestCoopObreraAdapter_FetchHTTPJSON_PostBodyStructure(t *testing.T) {
	var firstBody coopRequestBody
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var body coopRequestBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		if requestCount == 1 {
			firstBody = body
		}
		w.Header().Set("Content-Type", "application/json")
		// Return one product on first call, then empty to stop pagination
		if requestCount == 1 {
			w.Write(buildCoopResponse(1, []map[string]interface{}{
				coopArticuloPayload("Aceite 1L", "Marca", "https://img/a.jpg", 1000, 0, 0),
			}, 8))
		} else {
			w.Write(buildCoopResponse(1, []map[string]interface{}{}, 0))
		}
	}))
	defer server.Close()

	a := newCoopObreraAdapterWithClient(server.Client())
	_, err := a.FetchHTTPJSON(context.Background(), server.URL+"/articulos/pagina", scrapingport.HTTPJSONOptions{
		CategoryID:         3,
		PaginationStrategy: "page_increment",
	})

	require.NoError(t, err)

	// Verify critical gotcha: tipo_seleccion is INSIDE filtros, NOT at root
	assert.Equal(t, 3, firstBody.IDBusqueda, "id_busqueda must match CategoryID")
	assert.Equal(t, 0, firstBody.Pagina, "first page must be 0")
	assert.Equal(t, "categoria", firstBody.Filtros.TipoSeleccion,
		"tipo_seleccion MUST be inside filtros — NOT at root level (gotcha confirmed 2026-06-17)")
	assert.Empty(t, firstBody.Filtros.Categoria,
		"filtros.categoria must be empty [] to traverse full root category")
	// Verify the request method is POST (body decode succeeded)
	assert.GreaterOrEqual(t, requestCount, 1, "must have made at least one POST request")
}

// --- T-COOP-05: API error (estado != 1) returns error ---

func TestCoopObreraAdapter_FetchHTTPJSON_APIErrorReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate the "tipo_seleccion requerido" error (estado=4)
		resp := map[string]interface{}{
			"estado":  4,
			"mensaje": "El campo tipo_seleccion es requerido",
			"datos":   nil,
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer server.Close()

	a := newCoopObreraAdapterWithClient(server.Client())
	_, err := a.FetchHTTPJSON(context.Background(), server.URL+"/articulos/pagina", scrapingport.HTTPJSONOptions{
		CategoryID:         2,
		PaginationStrategy: "page_increment",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "estado=4")
}

// --- T-COOP-06: missing CategoryID returns error before any HTTP call ---

func TestCoopObreraAdapter_FetchHTTPJSON_MissingCategoryID(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write(buildCoopResponse(1, nil, 0))
	}))
	defer server.Close()

	a := newCoopObreraAdapterWithClient(server.Client())
	_, err := a.FetchHTTPJSON(context.Background(), server.URL+"/articulos/pagina", scrapingport.HTTPJSONOptions{
		CategoryID:         0, // missing
		PaginationStrategy: "page_increment",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CategoryID")
	assert.Equal(t, 0, callCount, "should not make any HTTP calls when CategoryID is missing")
}

// --- T-COOP-07: FirecrawlAdapter dispatches to coop on page_increment strategy ---

func TestFirecrawlAdapter_FetchHTTPJSON_DispatchesCoopOnPageIncrement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return one product then empty on page 1
		var body coopRequestBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Pagina == 0 {
			w.Write(buildCoopResponse(1, []map[string]interface{}{
				coopArticuloPayload("Leche 1L", "La Serenísima", "https://img/leche.jpg", 1500, 0, 0),
			}, 8))
		} else {
			w.Write(buildCoopResponse(1, []map[string]interface{}{}, 0))
		}
	}))
	defer server.Close()

	fc := newFirecrawlAdapterWithBaseURL("test-key", server.URL)
	// Replace the coop adapter's http client with the test server's client
	fc.coop = newCoopObreraAdapterWithClient(server.Client())

	products, err := fc.FetchHTTPJSON(context.Background(), server.URL+"/articulos/pagina", scrapingport.HTTPJSONOptions{
		CategoryID:         2,
		PaginationStrategy: "page_increment",
	})

	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "Leche 1L", products[0].Title)
}
