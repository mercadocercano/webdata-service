//go:build integration

// TEST-ID: T-INT-06
// Product controller E2E integration test.
// Seeds a source + products via UpsertProductsUseCase, then exercises:
//   GET /webdata/api/v1/products/
//   GET /webdata/api/v1/products/:id
//   GET /webdata/api/v1/products/:id/price-history
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
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
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
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { db.Close() })
	return db
}

type testModules struct {
	srv      *httptest.Server
	upsertUC *productusecase.UpsertProductsUseCase
}

func newTestModules(t *testing.T, db *sql.DB) testModules {
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
	return testModules{srv: srv, upsertUC: productModule.UpsertUC}
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

func doGet(t *testing.T, client *http.Client, url, token, tenantID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// ─── T-INT-06: Product controller E2E ────────────────────────────────────

func TestProductController_E2E_ListGetPriceHistory(t *testing.T) {
	db := testDB(t)
	modules := newTestModules(t, db)
	tenantID := uuid.New()
	t.Cleanup(func() { cleanupProducts(t, db, tenantID) })

	ctx := context.Background()
	sourceID := seedSource(t, db, tenantID)
	token := makeTestJWT(t, tenantID)

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
	baseProducts := modules.srv.URL + "/webdata/api/v1/products"

	// ── 1. List products ───────────────────────────────────────────────────
	resp := doGet(t, client, baseProducts+"/", token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products should return 200")

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data, ok := listResp["data"].([]any)
	require.True(t, ok, "response should have data array")
	assert.Len(t, data, 2, "should return 2 products")

	firstProduct := data[0].(map[string]any)
	productID, ok := firstProduct["id"].(string)
	require.True(t, ok, "product should have id")

	// ── 2. Get product by ID ───────────────────────────────────────────────
	resp = doGet(t, client, baseProducts+"/"+productID, token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id should return 200")

	var getResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getResp))
	resp.Body.Close()

	product := getResp["data"].(map[string]any)
	assert.NotEmpty(t, product["title"], "product should have title")

	// ── 3. Get price history ───────────────────────────────────────────────
	resp = doGet(t, client, baseProducts+"/"+productID+"/price-history", token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products/:id/price-history should return 200")
	resp.Body.Close()

	// ── 4. Filter by category ──────────────────────────────────────────────
	resp = doGet(t, client, baseProducts+"/?category=almacen", token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products?category= should return 200")

	var filtered map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&filtered))
	resp.Body.Close()

	filteredData := filtered["data"].([]any)
	assert.Len(t, filteredData, 2, "both products are in almacen category")

	// ── 5. Filter by price range ───────────────────────────────────────────
	resp = doGet(t, client, baseProducts+"/?max_price=1400", token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /products?max_price= should return 200")

	var priceFiltered map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&priceFiltered))
	resp.Body.Close()

	priceFilteredData := priceFiltered["data"].([]any)
	assert.Len(t, priceFilteredData, 1, "only 1 product is priced below 1400")

	// ── 6. Unknown product returns 404 ────────────────────────────────────
	resp = doGet(t, client, baseProducts+"/"+uuid.New().String(), token, tenantID.String())
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
	token := makeTestJWT(t, tenantID)
	client := modules.srv.Client()
	baseProducts := modules.srv.URL + "/webdata/api/v1/products"

	// First upsert — initial price
	price1 := 2000.00
	_, err := modules.upsertUC.Execute(ctx, tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price1, URL: "https://test.example.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	// Second upsert — price change triggers a new price record
	price2 := 1850.00
	_, err = modules.upsertUC.Execute(ctx, tenantID, sourceID, nil, []scrapeport.RawProduct{
		{Title: "Smart TV 43\"", Price: &price2, URL: "https://test.example.com/tv", Category: "electronica"},
	})
	require.NoError(t, err)

	// List to get product ID
	resp := doGet(t, client, baseProducts+"/", token, tenantID.String())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listResp))
	resp.Body.Close()

	data := listResp["data"].([]any)
	require.Len(t, data, 1)
	productID := data[0].(map[string]any)["id"].(string)

	// Get price history — should have records after price change
	resp = doGet(t, client, baseProducts+"/"+productID+"/price-history", token, tenantID.String())
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var histResp map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&histResp))
	resp.Body.Close()

	history := histResp["data"].([]any)
	assert.GreaterOrEqual(t, len(history), 1, "price history should have at least 1 record")
}
