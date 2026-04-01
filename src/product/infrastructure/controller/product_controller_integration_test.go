//go:build integration

// TEST-ID: T-INT-06
// Product controller E2E integration test.
// Seeds a source + products via UpsertProductsUseCase, then exercises:
//   GET /api/v1/products/
//   GET /api/v1/products/:id
//   GET /api/v1/products/:id/price-history
//
// Requires a running PostgreSQL with migrated webdata schema.
// Set WEBDATA_TEST_DB_URL or individual DB_* env vars.
// Run: go test -tags=integration ./src/product/infrastructure/controller/...
package controller_test

import (
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
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
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
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { db.Close() })
	return db
}

type testModules struct {
	srv        *httptest.Server
	upsertUC   interface {
		Execute(ctx context.Context, tenantID, sourceID uuid.UUID, jobID *uuid.UUID, raw []scrapeport.RawProduct) (int, error)
	}
}

func newTestModules(t *testing.T, db *sql.DB) testModules {
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
	return testModules{srv: srv, upsertUC: productModule.UpsertUC}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func cleanupProducts(t *testing.T, db *sql.DB, tenantID uuid.UUID) {
	t.Helper()
	_, _ = db.ExecContext(context.Background(), "DELETE FROM webdata_price_history WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(context.Background(), "DELETE FROM webdata_products WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(context.Background(), "DELETE FROM webdata_sources WHERE tenant_id = $1", tenantID)
}

// seedSource inserts a minimal source row directly for test isolation.
func seedSource(t *testing.T, db *sql.DB, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO webdata_sources
			(id, tenant_id, name, base_url, category, source_type, priority, tier, firecrawl_method, is_active, health_score)
		VALUES
			($1, $2, 'Integration Test Source', 'https://test.example.com', 'supermercado', 'ecommerce', 'high', 1, 'extract', true, 1.00)
	`, id, tenantID)
	require.NoError(t, err)
	return id
}

// ─── T-INT-06: Product controller E2E ────────────────────────────────────

func TestProductController_E2E_ListGetPriceHistory(t *testing.T) {
	db := testDB(t)
	modules := newTestModules(t, db)
	tenantID := uuid.New()
	t.Cleanup(func() { cleanupProducts(t, db, tenantID) })

	ctx := context.Background()
	sourceID := seedSource(t, db, tenantID)

	price1 := 1500.00
	price2 := 1350.00
	products := []scrapeport.RawProduct{
		{
			Title:    "Arroz Largo Fino x 1kg",
			Price:    &price1,
			URL:      "https://test.example.com/arroz",
			Brand:    "Molinos",
			Category: "almacen",
			SKU:      "ARR-001",
		},
		{
			Title:    "Aceite Girasol x 900ml",
			Price:    &price2,
			URL:      "https://test.example.com/aceite",
			Brand:    "Cocinero",
			Category: "almacen",
			SKU:      "ACE-001",
		},
	}

	saved, err := modules.upsertUC.Execute(ctx, tenantID, sourceID, nil, products)
	require.NoError(t, err)
	assert.Equal(t, 2, saved, "should have upserted 2 products")

	client := modules.srv.Client()
	baseProducts := modules.srv.URL + "/api/v1/products"

	// ── 1. List products ───────────────────────────────────────────────────
	req, _ := http.NewRequest(http.MethodGet, baseProducts+"/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products should return 200")

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data, ok := listResp["data"].([]any)
	require.True(t, ok, "response should have data array")
	assert.Len(t, data, 2, "should return 2 products")

	// Extract first product ID for subsequent tests
	firstProduct := data[0].(map[string]any)
	productID, ok := firstProduct["id"].(string)
	require.True(t, ok, "product should have id")

	// ── 2. Get product by ID ───────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/"+productID, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id should return 200")

	var product map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&product))
	resp.Body.Close()
	assert.NotEmpty(t, product["title"], "product should have title")
	assert.NotNil(t, product["current_price"], "product should have current_price")

	// ── 3. Get price history ───────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/"+productID+"/price-history", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id/price-history should return 200")
	resp.Body.Close()

	// ── 4. Filter by category ──────────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/?category=almacen", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products?category= should return 200")

	var filtered map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&filtered))
	resp.Body.Close()

	filteredData := filtered["data"].([]any)
	assert.Len(t, filteredData, 2, "both products are in almacen category")

	// ── 5. Filter by price range ───────────────────────────────────────────
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/?max_price=1400", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products?max_price= should return 200")

	var priceFiltered map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&priceFiltered))
	resp.Body.Close()

	priceFilteredData := priceFiltered["data"].([]any)
	assert.Len(t, priceFilteredData, 1, "only 1 product is priced below 1400")

	// ── 6. Get nonexistent product returns 404 ────────────────────────────
	nonexistentID := uuid.New()
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/"+nonexistentID.String(), nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "unknown product should return 404")
	resp.Body.Close()
}

func TestProductController_E2E_PriceHistoryTracking(t *testing.T) {
	db := testDB(t)
	modules := newTestModules(t, db)
	tenantID := uuid.New()
	t.Cleanup(func() { cleanupProducts(t, db, tenantID) })

	ctx := context.Background()
	sourceID := seedSource(t, db, tenantID)

	client := modules.srv.Client()
	baseProducts := modules.srv.URL + "/api/v1/products"

	// First upsert — initial price
	price1 := 2000.00
	_, err := modules.upsertUC.Execute(ctx, tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price1, URL: "https://test.example.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	// Second upsert — price change (should create price record)
	price2 := 1850.00
	_, err = modules.upsertUC.Execute(ctx, tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price2, URL: "https://test.example.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	// List to get product ID
	req, _ := http.NewRequest(http.MethodGet, baseProducts+"/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err := client.Do(req)
	require.NoError(t, err)
	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data := listResp["data"].([]any)
	require.Len(t, data, 1)
	productID := data[0].(map[string]any)["id"].(string)

	// Get price history — should have at least 2 records
	req, _ = http.NewRequest(http.MethodGet, baseProducts+"/"+productID+"/price-history", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var history []any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&history))
	resp.Body.Close()

	assert.GreaterOrEqual(t, len(history), 2, "price history should have at least 2 records after price change")
}
