//go:build integration

// TEST-ID: T-INT-05
// Source controller E2E integration test.
// Requires a running PostgreSQL with migrated webdata schema.
// Set WEBDATA_TEST_DB_URL or individual DB_* env vars.
// Run: go test -tags=integration ./src/source/infrastructure/controller/...
package controller_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mercadocercano/webdata-service/src/api"
	"github.com/mercadocercano/webdata-service/src/scraping/infrastructure/adapter"
	scrapingconfig "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/config"
	productconfig "github.com/mercadocercano/webdata-service/src/product/infrastructure/config"
	sourceconfig "github.com/mercadocercano/webdata-service/src/source/infrastructure/config"
	statsconfig "github.com/mercadocercano/webdata-service/src/stats/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	connStr := os.Getenv("WEBDATA_TEST_DB_URL")
	if connStr == "" {
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("DB_NAME", "webdata_test"),
		)
	}
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err, "open test DB")
	require.NoError(t, db.Ping(), "ping test DB")
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	productModule := productconfig.NewProductModule(db)
	sourceModule := sourceconfig.NewSourceModule(db, nil)
	firecrawl := adapter.NewFirecrawlAdapter("test-key-noop")
	scrapingModule := scrapingconfig.NewScrapingModule(db, sourceModule.Repo, firecrawl, productModule.UpsertUC)
	sourceModule = sourceconfig.NewSourceModule(db, scrapingModule.Repo)
	statsModule := statsconfig.NewStatsModule(sourceModule.Repo, scrapingModule.Repo, productModule.Repo)

	router := api.NewRouter(
		sourceModule.Controller,
		scrapingModule.Controller,
		productModule.Controller,
		statsModule.Controller,
	)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func newTenantID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	return id
}

func cleanupSources(t *testing.T, db *sql.DB, tenantID uuid.UUID) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"DELETE FROM webdata_sources WHERE tenant_id = $1", tenantID)
	require.NoError(t, err)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── T-INT-05: Source controller E2E ─────────────────────────────────────

func TestSourceController_E2E_CreateListGetUpdateDeleteTrigger(t *testing.T) {
	db := testDB(t)
	srv := newTestServer(t, db)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupSources(t, db, tenantID) })

	client := srv.Client()
	base := srv.URL + "/api/v1/sources"

	// ── 1. Create ──────────────────────────────────────────────────────────
	createBody := map[string]any{
		"name":             "Test Source NEA",
		"base_url":         "https://example.com",
		"category":         "supermercado",
		"source_type":      "ecommerce",
		"city":             "Posadas",
		"priority":         "high",
		"tier":             1,
		"firecrawl_method": "extract",
		"cron_expression":  "0 6 * * 1",
		"notes":            "integration test source",
	}
	body, _ := json.Marshal(createBody)
	req, _ := http.NewRequest(http.MethodPost, base+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST /sources should return 201")

	var created map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	resp.Body.Close()

	sourceID, ok := created["id"].(string)
	require.True(t, ok, "response should contain id")

	// ── 2. List ────────────────────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, base+"/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources should return 200")

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data, ok := listResp["data"].([]any)
	require.True(t, ok, "response should have data array")
	assert.Len(t, data, 1, "should list 1 source for this tenant")

	// ── 3. Get by ID ───────────────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, base+"/"+sourceID, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources/:id should return 200")

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	resp.Body.Close()
	assert.Equal(t, "Test Source NEA", got["name"])

	// ── 4. Update ──────────────────────────────────────────────────────────
	updateBody := map[string]any{"notes": "updated via integration test"}
	body, _ = json.Marshal(updateBody)
	req, _ = http.NewRequest(http.MethodPatch, base+"/"+sourceID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PATCH /sources/:id should return 200")

	var updated map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	resp.Body.Close()
	assert.Equal(t, "updated via integration test", updated["notes"])

	// ── 5. Trigger scrape ──────────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodPost, base+"/"+sourceID+"/trigger", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	// Trigger creates a job — expects 201 or 422 (if Firecrawl key is noop)
	assert.True(t,
		resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusUnprocessableEntity,
		"POST /sources/:id/trigger should return 201 or 422, got %d", resp.StatusCode)
	resp.Body.Close()

	// ── 6. Delete ──────────────────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodDelete, base+"/"+sourceID, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE /sources/:id should return 204")
	resp.Body.Close()

	// ── 7. Verify deleted (soft-delete: should return not found) ───────────
	req, _ = http.NewRequest(http.MethodGet, base+"/"+sourceID, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "GET after delete should return 404")
	resp.Body.Close()
}

func TestSourceController_E2E_TenantIsolation(t *testing.T) {
	db := testDB(t)
	srv := newTestServer(t, db)

	tenantA := newTenantID(t)
	tenantB := newTenantID(t)
	t.Cleanup(func() {
		cleanupSources(t, db, tenantA)
		cleanupSources(t, db, tenantB)
	})

	client := srv.Client()
	base := srv.URL + "/api/v1/sources"

	// Create source for tenant A
	body, _ := json.Marshal(map[string]any{
		"name": "Tenant A Source", "base_url": "https://a.com",
		"category": "supermercado", "source_type": "ecommerce", "tier": 1, "priority": "high",
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantA.String())
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Tenant B should see empty list
	req, _ = http.NewRequest(http.MethodGet, base+"/", nil)
	req.Header.Set("X-Tenant-ID", tenantB.String())
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data := listResp["data"].([]any)
	assert.Empty(t, data, "tenant B should see no sources from tenant A")
}

func TestSourceController_E2E_MissingTenantHeader(t *testing.T) {
	db := testDB(t)
	srv := newTestServer(t, db)
	client := srv.Client()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/sources/", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "missing X-Tenant-ID should return 400")
	resp.Body.Close()
}
