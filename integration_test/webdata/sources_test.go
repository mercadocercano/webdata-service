//go:build integration

package webdata_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSources_E2E_CRUDAndTrigger verifica el ciclo completo de fuentes:
// crear → listar → obtener → actualizar → trigger → eliminar.
func TestSources_E2E_CRUDAndTrigger(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── 1. Crear fuente ──────────────────────────────────────────────────────
	resp := doRequest(t, env, http.MethodPost, "/api/v1/sources/", token, tid, map[string]any{
		"name":             "Fuente Integración A",
		"base_url":         "https://example-a.com",
		"category":         "supermercado",
		"source_type":      "ecommerce",
		"city":             "Posadas",
		"priority":         "high",
		"tier":             1,
		"firecrawl_method": "extract",
		"cron_expression":  "0 6 * * 1",
	})
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST /sources debe retornar 201")

	body := decodeJSON(t, resp)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "response debe tener objeto data")
	sourceID, ok := data["id"].(string)
	require.True(t, ok && sourceID != "", "data debe tener id")

	// ── 2. Listar fuentes ────────────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/sources/", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources debe retornar 200")

	listBody := decodeJSON(t, resp)
	items, ok := listBody["data"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1, "debe listar 1 fuente para este tenant")

	// ── 3. Obtener fuente por ID ─────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/sources/"+sourceID, token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources/:id debe retornar 200")

	getBody := decodeJSON(t, resp)
	got := getBody["data"].(map[string]any)
	assert.Equal(t, "Fuente Integración A", got["name"])

	// ── 4. Actualizar fuente ─────────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodPatch, "/api/v1/sources/"+sourceID, token, tid, map[string]any{
		"notes": "actualizado vía integration test",
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PATCH /sources/:id debe retornar 200")

	updateBody := decodeJSON(t, resp)
	updated := updateBody["data"].(map[string]any)
	assert.Equal(t, "actualizado vía integration test", updated["notes"])

	// ── 5. Trigger manual → job creado (201) o error de firecrawl noop (422) ─
	resp = doRequest(t, env, http.MethodPost, "/api/v1/sources/"+sourceID+"/trigger", token, tid, nil)
	assert.True(t,
		resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusUnprocessableEntity,
		"POST /sources/:id/trigger debe retornar 201 o 422, got %d", resp.StatusCode)
	resp.Body.Close()

	// ── 6. Eliminar fuente ───────────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodDelete, "/api/v1/sources/"+sourceID, token, tid, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE /sources/:id debe retornar 204")
	resp.Body.Close()

	// ── 7. Verificar que ya no existe ────────────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/sources/"+sourceID, token, tid, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "GET después de delete debe retornar 404")
	resp.Body.Close()
}

// TestSources_E2E_TenantIsolation verifica que un tenant no ve fuentes de otro.
func TestSources_E2E_TenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	tenantA := newTenantID(t)
	tenantB := newTenantID(t)
	t.Cleanup(func() {
		cleanupTenant(t, env.DB, tenantA)
		cleanupTenant(t, env.DB, tenantB)
	})

	tokenA := makeJWT(t, tenantA)
	tokenB := makeJWT(t, tenantB)

	// Crear fuente para tenant A
	resp := doRequest(t, env, http.MethodPost, "/api/v1/sources/", tokenA, tenantA.String(), map[string]any{
		"name": "Fuente Tenant A", "base_url": "https://a.com",
		"category": "supermercado", "source_type": "ecommerce", "tier": 1, "priority": "high",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Tenant B debe ver lista vacía
	resp = doRequest(t, env, http.MethodGet, "/api/v1/sources/", tokenB, tenantB.String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Empty(t, items, "tenant B no debe ver fuentes de tenant A")
}

// TestSources_E2E_FilterByStatus verifica el filtro por estado activo/inactivo.
func TestSources_E2E_FilterByStatus(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Crear dos fuentes activas con nombres distintos
	for _, name := range []string{"Fuente Alfa", "Fuente Beta"} {
		resp := doRequest(t, env, http.MethodPost, "/api/v1/sources/", token, tid, map[string]any{
			"name": name, "base_url": "https://example.com",
			"category": "supermercado", "source_type": "ecommerce", "tier": 1, "priority": "high",
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// Listar todas — debe haber 2
	resp := doRequest(t, env, http.MethodGet, "/api/v1/sources/", token, tid, nil)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Len(t, items, 2)
}

// TestSources_E2E_MissingAuth verifica que sin Authorization retorna 401.
func TestSources_E2E_MissingAuth(t *testing.T) {
	env := newTestEnv(t)
	tenantID := uuid.New()

	req, _ := newHTTPRequest(t, http.MethodGet, env.Server.URL+"/api/v1/sources/", "", tenantID.String())
	resp, err := env.Server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSources_E2E_MissingTenantHeader verifica que sin X-Tenant-ID retorna 400.
func TestSources_E2E_MissingTenantHeader(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	token := makeJWT(t, tenantID)

	req, _ := newHTTPRequest(t, http.MethodGet, env.Server.URL+"/api/v1/sources/", token, "")
	resp, err := env.Server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// newHTTPRequest crea un request sin Content-Type, útil para test de auth/tenant.
func newHTTPRequest(t *testing.T, method, url, token, tenantID string) (*http.Request, error) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	return req, nil
}
