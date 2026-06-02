//go:build integration

// Package webdata_test contiene tests de integración para webdata-service.
// Usa TestContainers para levantar un PostgreSQL real y verificar el comportamiento completo
// del stack HTTP → Handler → UseCase → Repository → DB.
//
// Ejecutar con: go test -tags=integration ./integration_test/webdata/... -v
package webdata_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mercadocercano/webdata-service/src/api"
	enrichconfig "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/config"
	scraperadapter "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/adapter"
	scrapingconfig "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/config"
	productconfig "github.com/mercadocercano/webdata-service/src/product/infrastructure/config"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceconfig "github.com/mercadocercano/webdata-service/src/source/infrastructure/config"
	statsconfig "github.com/mercadocercano/webdata-service/src/stats/infrastructure/config"
)

const testJWTSecret = "test-jwt-secret-webdata-integration"

// testEnv agrupa el servidor HTTP, DB y use cases auxiliares para los tests.
type testEnv struct {
	Server   *httptest.Server
	DB       *sql.DB
	UpsertUC *productusecase.UpsertProductsUseCase
}

// newTestEnv levanta un contenedor PostgreSQL, ejecuta migraciones
// y retorna un testEnv listo para recibir requests HTTP.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("webdata_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("error starting postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("warn: failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("error getting connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("error opening database: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("error pinging database: %v", err)
	}

	runMigrations(t, db)

	t.Setenv("JWT_SECRET", testJWTSecret)

	productModule := productconfig.NewProductModule(db)
	sourceModule := sourceconfig.NewSourceModule(db, nil)
	firecrawl := scraperadapter.NewFirecrawlAdapter("test-key-noop")
	scrapingModule := scrapingconfig.NewScrapingModule(db, sourceModule.Repo, firecrawl, productModule.UpsertUC)
	sourceModule = sourceconfig.NewSourceModule(db, scrapingModule.Repo)
	statsModule := statsconfig.NewStatsModule(sourceModule.Repo, scrapingModule.Repo, productModule.Repo)
	btProxyCtrl := productcontroller.NewBusinessTypeProxyController()
	enrichModule := enrichconfig.NewEnrichmentModule(db)

	router := api.NewRouter(
		sourceModule.Controller,
		scrapingModule.Controller,
		productModule.Controller,
		statsModule.Controller,
		btProxyCtrl,
		enrichModule.Handler,
	)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testEnv{
		Server:   srv,
		DB:       db,
		UpsertUC: productModule.UpsertUC,
	}
}

// ── JWT helpers ──────────────────────────────────────────────────────────────

func makeJWT(t *testing.T, tenantID uuid.UUID) string {
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

// ── HTTP request helpers ─────────────────────────────────────────────────────

func doRequest(t *testing.T, env *testEnv, method, path, token, tenantID string, body any) *http.Response {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err)
	}

	var req *http.Request
	var err error
	if bodyBytes != nil {
		req, err = http.NewRequest(method, env.Server.URL+path, bytes.NewReader(bodyBytes))
	} else {
		req, err = http.NewRequest(method, env.Server.URL+path, nil)
	}
	require.NoError(t, err)

	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := env.Server.Client().Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// ── Seed helpers ─────────────────────────────────────────────────────────────

func seedSource(t *testing.T, db *sql.DB, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	// El nombre es único por tenant, se usa el UUID para evitar colisiones
	name := "Integration Test Source " + id.String()[:8]
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO webdata_sources
			(id, tenant_id, name, base_url, category, source_type, priority, tier, firecrawl_method, is_active, health_score)
		VALUES
			($1, $2, $3, 'https://test.example.com', 'supermercado', 'ecommerce', 'high', 1, 'extract', true, 1.00)
	`, id, tenantID, name)
	require.NoError(t, err)
	return id
}

func seedJob(t *testing.T, db *sql.DB, tenantID, sourceID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO webdata_scraping_jobs
			(id, tenant_id, source_id, status, trigger_type, page, retry_count, max_retries, created_at)
		VALUES
			($1, $2, $3, $4, 'manual', 1, 0, 3, NOW())
	`, id, tenantID, sourceID, status)
	require.NoError(t, err)
	return id
}

func seedProducts(t *testing.T, env *testEnv, tenantID, sourceID uuid.UUID, count int) []string {
	t.Helper()
	products := make([]scrapeport.RawProduct, count)
	for i := 0; i < count; i++ {
		price := float64(1000 + i*100)
		products[i] = scrapeport.RawProduct{
			Title:    fmt.Sprintf("Producto Test %d", i+1),
			Price:    &price,
			URL:      fmt.Sprintf("https://test.example.com/product-%d", i+1),
			Category: "almacen",
			SKU:      fmt.Sprintf("SKU-%03d", i+1),
		}
	}
	saved, err := env.UpsertUC.Execute(context.Background(), tenantID, sourceID, nil, products)
	require.NoError(t, err)
	require.Equal(t, count, saved)
	return nil
}

// ── Cleanup helpers ──────────────────────────────────────────────────────────

func cleanupTenant(t *testing.T, db *sql.DB, tenantID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, _ = db.ExecContext(ctx, "DELETE FROM webdata_price_history WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM webdata_product_business_types WHERE product_id IN (SELECT id FROM webdata_products WHERE tenant_id = $1)", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM webdata_products WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM webdata_scraping_jobs WHERE tenant_id = $1", tenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM webdata_sources WHERE tenant_id = $1", tenantID)
}

// ── Migrations ───────────────────────────────────────────────────────────────

func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	migrationsDir := findMigrationsDir(t)

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("error reading migrations dir %s: %v", migrationsDir, err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		name := entry.Name()
		if isDownMigration(name) {
			continue
		}
		sqlFiles = append(sqlFiles, filepath.Join(migrationsDir, name))
	}
	sort.Strings(sqlFiles)

	for _, f := range sqlFiles {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("error reading migration %s: %v", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Logf("warn: migration %s may have failed (idempotent ok): %v", filepath.Base(f), err)
		}
	}
}

func findMigrationsDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../migrations",
		"../migrations",
		"migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	t.Fatalf("could not find migrations directory")
	return ""
}

func isDownMigration(name string) bool {
	return len(name) > 9 && name[len(name)-9:] == ".down.sql"
}
