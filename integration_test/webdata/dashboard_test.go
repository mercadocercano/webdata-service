//go:build integration

package webdata_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDashboard_E2E_StatsMatchSeededData verifica que las métricas del dashboard
// reflejan correctamente los datos creados (fuentes, jobs, productos).
func TestDashboard_E2E_StatsMatchSeededData(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	// Crear 2 fuentes activas
	source1 := seedSource(t, env.DB, tenantID)
	source2 := seedSource(t, env.DB, tenantID)
	_ = source2

	// Crear jobs en distintos estados
	seedJob(t, env.DB, tenantID, source1, "completed")
	seedJob(t, env.DB, tenantID, source1, "completed")
	seedJob(t, env.DB, tenantID, source1, "failed")
	seedJob(t, env.DB, tenantID, source1, "pending")

	// Crear 5 productos vía upsert
	seedProducts(t, env, tenantID, source1, 5)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── GET /api/v1/stats → dashboard stats ──────────────────────────────────
	resp := doRequest(t, env, http.MethodGet, "/api/v1/stats/", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /stats debe retornar 200")

	body := decodeJSON(t, resp)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok, "respuesta debe contener campo data")

	// Verificar campos de fuentes
	assert.Equal(t, float64(2), data["total_sources"], "debe contar 2 fuentes")
	assert.Equal(t, float64(2), data["active_sources"], "ambas fuentes deben ser activas")

	// Verificar campos de jobs
	assert.Equal(t, float64(4), data["total_jobs"], "debe contar 4 jobs")
	assert.Equal(t, float64(2), data["completed_jobs"], "debe contar 2 completados")
	assert.Equal(t, float64(1), data["failed_jobs"], "debe contar 1 fallido")
	assert.Equal(t, float64(1), data["pending_jobs"], "debe contar 1 pendiente")
	assert.Equal(t, float64(0), data["running_jobs"], "debe contar 0 running")

	// Verificar campo de productos
	assert.Equal(t, float64(5), data["total_products"], "debe contar 5 productos")
}

// TestDashboard_E2E_EmptyTenantReturnsZeros verifica que un tenant sin datos retorna ceros.
func TestDashboard_E2E_EmptyTenantReturnsZeros(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	// No cleanup needed — no data created

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	resp := doRequest(t, env, http.MethodGet, "/api/v1/stats/", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	data := body["data"].(map[string]any)

	assert.Equal(t, float64(0), data["total_sources"])
	assert.Equal(t, float64(0), data["total_jobs"])
	assert.Equal(t, float64(0), data["total_products"])
}

// TestDashboard_E2E_TenantIsolation verifica que las estadísticas son por tenant.
func TestDashboard_E2E_TenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	tenantA := newTenantID(t)
	tenantB := newTenantID(t)
	t.Cleanup(func() {
		cleanupTenant(t, env.DB, tenantA)
		cleanupTenant(t, env.DB, tenantB)
	})

	// Crear datos solo para tenant A
	sourceA := seedSource(t, env.DB, tenantA)
	seedJob(t, env.DB, tenantA, sourceA, "completed")
	seedProducts(t, env, tenantA, sourceA, 10)

	tokenB := makeJWT(t, tenantB)

	// Tenant B debe ver ceros
	resp := doRequest(t, env, http.MethodGet, "/api/v1/stats/", tokenB, tenantB.String(), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeJSON(t, resp)
	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total_sources"], "tenant B no debe ver fuentes de tenant A")
	assert.Equal(t, float64(0), data["total_jobs"], "tenant B no debe ver jobs de tenant A")
	assert.Equal(t, float64(0), data["total_products"], "tenant B no debe ver productos de tenant A")
}

// TestDashboard_E2E_SourceStatsEndpoint verifica el endpoint de stats por fuente.
func TestDashboard_E2E_SourceStatsEndpoint(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	seedSource(t, env.DB, tenantID)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	resp := doRequest(t, env, http.MethodGet, "/api/v1/stats/sources", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /stats/sources debe retornar 200")

	body := decodeJSON(t, resp)
	data, ok := body["data"].([]any)
	require.True(t, ok, "respuesta debe tener array data")
	require.Len(t, data, 1, "debe retornar stats de la fuente creada")

	sourceStats := data[0].(map[string]any)
	assert.NotEmpty(t, sourceStats["source_id"])
	assert.NotEmpty(t, sourceStats["name"])
}
