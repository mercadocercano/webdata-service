//go:build integration

package webdata_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

// TestProducts_E2E_ListGetDelete verifica CRUD básico de productos.
func TestProducts_E2E_ListGetDelete(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedProducts(t, env, tenantID, sourceID, 3)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── 1. Listar productos ──────────────────────────────────────────────────
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products debe retornar 200")

	listBody := decodeJSON(t, resp)
	items, ok := listBody["data"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 3, "debe listar 3 productos")

	firstID := items[0].(map[string]any)["id"].(string)

	// ── 2. Obtener producto por ID ───────────────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/"+firstID, token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id debe retornar 200")

	getBody := decodeJSON(t, resp)
	product := getBody["data"].(map[string]any)
	assert.NotEmpty(t, product["title"])

	// ── 3. Eliminar producto ─────────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodDelete, "/api/v1/products/"+firstID, token, tid, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE /products/:id debe retornar 204")
	resp.Body.Close()

	// ── 4. Lista debe mostrar 2 después del soft-delete ──────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	afterDelBody := decodeJSON(t, resp)
	afterItems := afterDelBody["data"].([]any)
	assert.Len(t, afterItems, 2, "debe mostrar 2 después del soft-delete")

	// ── 5. Producto inexistente retorna 404 ──────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/"+newTenantID(t).String(), token, tid, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "producto inexistente debe retornar 404")
	resp.Body.Close()
}

// TestProducts_E2E_FilterBySourceAndCategory verifica filtros de lista.
func TestProducts_E2E_FilterBySourceAndCategory(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)

	// Seed 2 productos en "almacen" y 1 en "electronica"
	priceA := 500.0
	priceB := 800.0
	priceC := 1200.0
	_, err := env.UpsertUC.Execute(context.Background(), tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Arroz", Price: &priceA, URL: "https://t.com/arroz", Category: "almacen", SKU: "ARR-001"},
		{Title: "Aceite", Price: &priceB, URL: "https://t.com/aceite", Category: "almacen", SKU: "ACE-001"},
		{Title: "TV", Price: &priceC, URL: "https://t.com/tv", Category: "electronica", SKU: "TV-001"},
	})
	require.NoError(t, err)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Filtrar por category=almacen
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/?category=almacen", token, tid, nil)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Len(t, items, 2, "debe listar 2 productos de almacen")

	// Filtrar por category=electronica
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/?category=electronica", token, tid, nil)
	body = decodeJSON(t, resp)
	items = body["data"].([]any)
	assert.Len(t, items, 1, "debe listar 1 producto de electronica")

	// Filtrar por source_id
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/?source_id="+sourceID.String(), token, tid, nil)
	body = decodeJSON(t, resp)
	items = body["data"].([]any)
	assert.Len(t, items, 3, "debe listar 3 productos de la fuente")
}

// TestProducts_E2E_BulkDelete verifica la eliminación masiva.
func TestProducts_E2E_BulkDelete(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedProducts(t, env, tenantID, sourceID, 5)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Listar para obtener IDs
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	require.Len(t, items, 5)

	// Tomar 3 IDs para bulk-delete
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		ids[i] = items[i].(map[string]any)["id"].(string)
	}

	resp = doRequest(t, env, http.MethodPost, "/api/v1/products/bulk-delete", token, tid, map[string]any{
		"ids": ids,
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode, "POST /products/bulk-delete debe retornar 200")

	bulkBody := decodeJSON(t, resp)
	assert.Equal(t, float64(3), bulkBody["deleted"], "debe eliminar 3 productos")

	// Verificar que quedan 2
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	afterBody := decodeJSON(t, resp)
	afterItems := afterBody["data"].([]any)
	assert.Len(t, afterItems, 2, "deben quedar 2 productos")
}

// TestProducts_E2E_AssignBusinessType verifica la asignación individual de business type.
func TestProducts_E2E_AssignBusinessType(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedProducts(t, env, tenantID, sourceID, 1)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Obtener el ID del producto
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	require.Len(t, items, 1)
	productID := items[0].(map[string]any)["id"].(string)

	// Asignar business type — el handler espera {business_types: [{code, name}]}
	resp = doRequest(t, env, http.MethodPut, "/api/v1/products/"+productID+"/business-types", token, tid, map[string]any{
		"business_types": []map[string]string{
			{"code": "supermercado", "name": "Supermercado"},
		},
	})
	assert.Equal(t, http.StatusNoContent, resp.StatusCode,
		"PUT /products/:id/business-types debe retornar 204")
	resp.Body.Close()
}

// TestProducts_E2E_BulkAssignBusinessType verifica la asignación masiva de business type.
func TestProducts_E2E_BulkAssignBusinessType(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedProducts(t, env, tenantID, sourceID, 5)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Obtener IDs de los 5 productos
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	require.Len(t, items, 5)

	ids := make([]string, 5)
	for i, item := range items {
		ids[i] = item.(map[string]any)["id"].(string)
	}

	// Bulk-assign business type — el handler espera {product_ids, code, name}
	resp = doRequest(t, env, http.MethodPost, "/api/v1/products/bulk-assign-business-type", token, tid, map[string]any{
		"product_ids": ids,
		"code":        "supermercado",
		"name":        "Supermercado",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"POST /products/bulk-assign-business-type debe retornar 200")

	assignBody := decodeJSON(t, resp)
	// La respuesta tiene {Assigned, Failed, Errors}
	assigned, ok := assignBody["Assigned"]
	require.True(t, ok, "debe retornar campo Assigned")
	assert.Equal(t, float64(5), assigned, "deben asignarse los 5 productos")
}

// TestProducts_E2E_PriceHistory verifica el historial de precios.
func TestProducts_E2E_PriceHistory(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)

	// Primer upsert — precio inicial
	price1 := 2000.0
	_, err := env.UpsertUC.Execute(context.Background(), tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price1, URL: "https://t.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	// Segundo upsert — cambio de precio
	price2 := 1750.0
	_, err = env.UpsertUC.Execute(context.Background(), tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price2, URL: "https://t.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Obtener ID del producto
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	require.Len(t, items, 1)
	productID := items[0].(map[string]any)["id"].(string)

	// Obtener historial de precios
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/"+productID+"/price-history", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id/price-history debe retornar 200")

	histBody := decodeJSON(t, resp)
	history, ok := histBody["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(history), 1, "debe haber al menos 1 registro de precio")
}

// TestProducts_E2E_TenantIsolation verifica que un tenant no ve productos de otro.
func TestProducts_E2E_TenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	tenantA := newTenantID(t)
	tenantB := newTenantID(t)
	t.Cleanup(func() {
		cleanupTenant(t, env.DB, tenantA)
		cleanupTenant(t, env.DB, tenantB)
	})

	sourceA := seedSource(t, env.DB, tenantA)
	seedProducts(t, env, tenantA, sourceA, 3)

	tokenB := makeJWT(t, tenantB)

	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", tokenB, tenantB.String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Empty(t, items, "tenant B no debe ver productos de tenant A")
}

// TestProducts_E2E_UpdateProduct verifica la actualización (PATCH) de un producto.
func TestProducts_E2E_UpdateProduct(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedProducts(t, env, tenantID, sourceID, 1)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Obtener producto
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	require.Len(t, items, 1)
	productID := items[0].(map[string]any)["id"].(string)

	// Bloquear el producto
	resp = doRequest(t, env, http.MethodPatch, "/api/v1/products/"+productID, token, tid, map[string]any{
		"is_blocked": true,
	})
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "PATCH debe retornar 204")
	resp.Body.Close()

	// Verificar que está bloqueado
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/"+productID, token, tid, nil)
	getBody := decodeJSON(t, resp)
	product := getBody["data"].(map[string]any)
	assert.Equal(t, true, product["is_blocked"], "el producto debe estar bloqueado")
}

// TestProducts_E2E_BulkDeleteEmpty verifica que bulk-delete con lista vacía retorna 400.
func TestProducts_E2E_BulkDeleteEmpty(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	resp := doRequest(t, env, http.MethodPost, "/api/v1/products/bulk-delete", token, tid, map[string]any{
		"ids": []string{},
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"bulk-delete con ids vacíos debe retornar 400")
	resp.Body.Close()
}

// TestProducts_E2E_Pagination verifica que la paginación funciona correctamente.
func TestProducts_E2E_Pagination(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)

	// 10 productos
	products := make([]scrapeport.RawProduct, 10)
	for i := 0; i < 10; i++ {
		price := float64((i + 1) * 100)
		products[i] = scrapeport.RawProduct{
			Title:    fmt.Sprintf("Producto Paginado %d", i+1),
			Price:    &price,
			URL:      fmt.Sprintf("https://t.com/pagprod-%d", i+1),
			Category: "almacen",
			SKU:      fmt.Sprintf("PP-%03d", i+1),
		}
	}
	_, err := env.UpsertUC.Execute(context.Background(), tenantID, sourceID, nil, products)
	require.NoError(t, err)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Total sin paginación — debe haber 10
	resp := doRequest(t, env, http.MethodGet, "/api/v1/products/", token, tid, nil)
	body := decodeJSON(t, resp)
	allItems := body["data"].([]any)
	assert.Len(t, allItems, 10, "debe listar los 10 productos sin paginación personalizada")

	// Paginación page_size=3, página 1 → 3 ítems
	resp = doRequest(t, env, http.MethodGet, "/api/v1/products/?page_size=3&page=1", token, tid, nil)
	pageBody := decodeJSON(t, resp)
	pageItems := pageBody["data"].([]any)
	assert.Len(t, pageItems, 3, "primera página debe tener 3 productos")

	// Metadatos de paginación correctos
	meta, ok := pageBody["meta"].(map[string]any)
	require.True(t, ok, "respuesta debe tener campo meta")
	assert.Equal(t, float64(10), meta["total"], "meta.total debe ser 10")
	assert.Equal(t, float64(3), meta["page_size"], "meta.page_size debe ser 3")
}
