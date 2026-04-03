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
	"time"

	"github.com/golang-jwt/jwt/v5"
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

const testJWTSecret = "test-jwt-secret-for-integration-tests"

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
	t.Setenv("JWT_SECRET", testJWTSecret)

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

func makeTestJWT(t *testing.T, tenantID uuid.UUID) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":   uuid.New().String(),
		"tenant_id": tenantID.String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
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

func doRequest(t *testing.T, client *http.Client, method, url, token, tenantID string, body []byte) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// ─── T-INT-05: Source controller E2E ─────────────────────────────────────

func TestSourceController_E2E_CreateListGetUpdateDeleteTrigger(t *testing.T) {
	db := testDB(t)
	srv := newTestServer(t, db)
	tenantID := newTenantID(t)
	t.Cleanup(func() { cleanupSources(t, db, tenantID) })

	client := srv.Client()
	token := makeTestJWT(t, tenantID)
	base := srv.URL + "/api/v1/sources"

	// ── 1. Create ──────────────────────────────────────────────────────────
	createBody, _ := json.Marshal(map[string]any{
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
	})
	resp := doRequest(t, client, http.MethodPost, base+"/", token, tenantID.String(), createBody)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST /sources should return 201")

	var createResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResp))
	resp.Body.Close()

	data, ok := createResp["data"].(map[string]any)
	require.True(t, ok, "response should have data object")
	sourceID, ok := data["id"].(string)
	require.True(t, ok, "data should contain id")

	// ── 2. List ────────────────────────────────────────────────────────────
	resp = doRequest(t, client, http.MethodGet, base+"/", token, tenantID.String(), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources should return 200")

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	items, ok := listResp["data"].([]any)
	require.True(t, ok, "response should have data array")
	assert.Len(t, items, 1, "should list 1 source for this tenant")

	// ── 3. Get by ID ───────────────────────────────────────────────────────
	resp = doRequest(t, client, http.MethodGet, base+"/"+sourceID, token, tenantID.String(), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /sources/:id should return 200")

	var getResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getResp))
	resp.Body.Close()

	got := getResp["data"].(map[string]any)
	assert.Equal(t, "Test Source NEA", got["name"])

	// ── 4. Update ──────────────────────────────────────────────────────────
	updateBody, _ := json.Marshal(map[string]any{"notes": "updated via integration test"})
	resp = doRequest(t, client, http.MethodPatch, base+"/"+sourceID, token, tenantID.String(), updateBody)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PATCH /sources/:id should return 200")

	var updateResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updateResp))
	resp.Body.Close()
	updated := updateResp["data"].(map[string]any)
	assert.Equal(t, "updated via integration test", updated["notes"])

	// ── 5. Trigger scrape ──────────────────────────────────────────────────
	resp = doRequest(t, client, http.MethodPost, base+"/"+sourceID+"/trigger", token, tenantID.String(), nil)
	// Trigger creates a job — 201 (success) or 422 (Firecrawl noop key)
	assert.True(t,
		resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusUnprocessableEntity,
		"POST /sources/:id/trigger should return 201 or 422, got %d", resp.StatusCode)
	resp.Body.Close()

	// ── 6. Delete ──────────────────────────────────────────────────────────
	resp = doRequest(t, client, http.MethodDelete, base+"/"+sourceID, token, tenantID.String(), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE /sources/:id should return 204")
	resp.Body.Close()

	// ── 7. Verify deleted → 404 ────────────────────────────────────────────
	resp = doRequest(t, client, http.MethodGet, base+"/"+sourceID, token, tenantID.String(), nil)
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
	tokenA := makeTestJWT(t, tenantA)
	tokenB := makeTestJWT(t, tenantB)
	base := srv.URL + "/api/v1/sources"

	// Create source for tenant A
	body, _ := json.Marshal(map[string]any{
		"name": "Tenant A Source", "base_url": "https://a.com",
		"category": "supermercado", "source_type": "ecommerce", "tier": 1, "priority": "high",
	})
	resp := doRequest(t, client, http.MethodPost, base+"/", tokenA, tenantA.String(), body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Tenant B should see empty list
	resp = doRequest(t, client, http.MethodGet, base+"/", tokenB, tenantB.String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

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
	tenantID := newTenantID(t)
	token := makeTestJWT(t, tenantID)

	// Request with auth but without X-Tenant-ID
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/sources/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "missing X-Tenant-ID should return 400")
	resp.Body.Close()
}

func TestSourceController_E2E_MissingAuthHeader(t *testing.T) {
	db := testDB(t)
	srv := newTestServer(t, db)
	client := srv.Client()
	tenantID := newTenantID(t)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/sources/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "missing Authorization should return 401")
	resp.Body.Close()
}
