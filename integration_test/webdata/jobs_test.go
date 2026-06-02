//go:build integration

package webdata_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobs_E2E_ListAndGet verifica la lista y obtención de jobs por ID.
func TestJobs_E2E_ListAndGet(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	jobID := seedJob(t, env.DB, tenantID, sourceID, "pending")

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── 1. Listar jobs ───────────────────────────────────────────────────────
	resp := doRequest(t, env, http.MethodGet, "/api/v1/jobs/", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /jobs debe retornar 200")

	listBody := decodeJSON(t, resp)
	items, ok := listBody["data"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1, "debe listar 1 job")

	// ── 2. Obtener job por ID ────────────────────────────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/jobs/"+jobID.String(), token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /jobs/:id debe retornar 200")

	getBody := decodeJSON(t, resp)
	job := getBody["data"].(map[string]any)
	assert.Equal(t, jobID.String(), job["id"])
	assert.Equal(t, "pending", job["status"])
}

// TestJobs_E2E_FilterByStatus verifica el filtro por status.
func TestJobs_E2E_FilterByStatus(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	seedJob(t, env.DB, tenantID, sourceID, "pending")
	seedJob(t, env.DB, tenantID, sourceID, "completed")
	seedJob(t, env.DB, tenantID, sourceID, "failed")

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Filtrar por pending
	resp := doRequest(t, env, http.MethodGet, "/api/v1/jobs/?status=pending", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Len(t, items, 1, "debe filtrar solo los jobs pending")

	// Filtrar por completed
	resp = doRequest(t, env, http.MethodGet, "/api/v1/jobs/?status=completed", token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body = decodeJSON(t, resp)
	items = body["data"].([]any)
	assert.Len(t, items, 1, "debe filtrar solo los jobs completed")
}

// TestJobs_E2E_CancelPendingJob verifica que un job pending puede cancelarse.
func TestJobs_E2E_CancelPendingJob(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	jobID := seedJob(t, env.DB, tenantID, sourceID, "pending")

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── 1. Cancelar job pending ──────────────────────────────────────────────
	resp := doRequest(t, env, http.MethodPost, "/api/v1/jobs/"+jobID.String()+"/cancel", token, tid, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "POST /jobs/:id/cancel debe retornar 204")
	resp.Body.Close()

	// ── 2. Verificar que el status cambió a cancelled ────────────────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/jobs/"+jobID.String(), token, tid, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	getBody := decodeJSON(t, resp)
	job := getBody["data"].(map[string]any)
	assert.Equal(t, "cancelled", job["status"], "el job debe estar cancelado")
}

// TestJobs_E2E_RetryFailedJob verifica que un job failed puede reintentarse.
func TestJobs_E2E_RetryFailedJob(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	failedJobID := seedJob(t, env.DB, tenantID, sourceID, "failed")

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// ── 1. Retry del job fallido ─────────────────────────────────────────────
	resp := doRequest(t, env, http.MethodPost, "/api/v1/jobs/"+failedJobID.String()+"/retry", token, tid, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST /jobs/:id/retry debe retornar 201")

	retryBody := decodeJSON(t, resp)
	newJobData := retryBody["data"].(map[string]any)
	newJobID, ok := newJobData["id"].(string)
	require.True(t, ok && newJobID != "")
	assert.NotEqual(t, failedJobID.String(), newJobID, "debe crear un nuevo job")
	assert.Equal(t, "pending", newJobData["status"], "el nuevo job debe estar pending")

	// ── 2. Verificar que hay 2 jobs en total (original + retry) ─────────────
	resp = doRequest(t, env, http.MethodGet, "/api/v1/jobs/", token, tid, nil)
	listBody := decodeJSON(t, resp)
	items := listBody["data"].([]any)
	assert.GreaterOrEqual(t, len(items), 2, "debe haber al menos 2 jobs (original + retry)")
}

// TestJobs_E2E_CannotCancelRunningJob verifica que un job running no puede cancelarse.
func TestJobs_E2E_CannotCancelRunningJob(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupTenant(t, env.DB, tenantID) })

	sourceID := seedSource(t, env.DB, tenantID)
	// Crear un job en estado running directamente en la DB
	runningJobID := seedJob(t, env.DB, tenantID, sourceID, "running")

	// Marcar como started en la DB
	_, err := env.DB.ExecContext(context.Background(),
		"UPDATE webdata_scraping_jobs SET started_at = NOW() WHERE id = $1", runningJobID)
	require.NoError(t, err)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	// Intentar cancelar job running debe fallar
	resp := doRequest(t, env, http.MethodPost, "/api/v1/jobs/"+runningJobID.String()+"/cancel", token, tid, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
		"cancelar un job running debe retornar 422")
	resp.Body.Close()
}

// TestJobs_E2E_GetUnknownJob verifica que un job inexistente retorna 404.
func TestJobs_E2E_GetUnknownJob(t *testing.T) {
	env := newTestEnv(t)
	tenantID := newTenantID(t)

	token := makeJWT(t, tenantID)
	tid := tenantID.String()

	resp := doRequest(t, env, http.MethodGet, "/api/v1/jobs/"+newTenantID(t).String(), token, tid, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "job inexistente debe retornar 404")
	resp.Body.Close()
}

// TestJobs_E2E_TenantIsolation verifica que un tenant no ve jobs de otro.
func TestJobs_E2E_TenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	tenantA := newTenantID(t)
	tenantB := newTenantID(t)
	t.Cleanup(func() {
		cleanupTenant(t, env.DB, tenantA)
		cleanupTenant(t, env.DB, tenantB)
	})

	sourceA := seedSource(t, env.DB, tenantA)
	seedJob(t, env.DB, tenantA, sourceA, "completed")

	tokenB := makeJWT(t, tenantB)

	// Tenant B no debe ver jobs de tenant A
	resp := doRequest(t, env, http.MethodGet, "/api/v1/jobs/", tokenB, tenantB.String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decodeJSON(t, resp)
	items := body["data"].([]any)
	assert.Empty(t, items, "tenant B no debe ver jobs de tenant A")
}
