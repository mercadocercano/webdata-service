//go:build ignore

// seed_national_sources seeds 3 national ecommerce sources into webdata_sources.
// Usage: go run scripts/seed_national_sources.go
// Env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, TENANT_ID
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type sourceRow struct {
	name             string
	baseURL          string
	category         string
	sourceType       string
	city             string
	priority         string
	tier             int
	firecrawlMethod  string
	cronExpression   string
	notes            string
	extractionSchema string
}

var nationalSources = []sourceRow{
	// ── Tier 1 — Scraping semanal ──────────────────────────────────────────
	{
		name:            "Blaisten",
		baseURL:         "https://www.blaisten.com/sanitarios?page=2",
		category:        "ferreteria_construccion",
		sourceType:      "ecommerce",
		city:            "Nacional",
		priority:        "medium",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 1", // lunes 06:00
		notes:           "VTEX store, extracción directa sin config especial. Paginación por query param ?page=N",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}`,
	},
	// ── Tier 2 — Scraping semanal, mayor complejidad ───────────────────────
	{
		name:            "Frávega",
		baseURL:         "https://www.fravega.com/electrodomesticos/",
		category:        "electronica",
		sourceType:      "ecommerce",
		city:            "Nacional",
		priority:        "medium",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 1", // lunes 06:00
		notes:           "SPA pesado, requiere waitFor >= 10000 y acciones de scroll. 9+ créditos por extract.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}`,
	},
	{
		name:            "Dexter",
		baseURL:         "https://www.dexter.com.ar/zapatillas",
		category:        "indumentaria",
		sourceType:      "ecommerce",
		city:            "Nacional",
		priority:        "medium",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 1", // lunes 06:00
		notes:           "Salesforce Commerce Cloud. URLs complejas con pipes funcionan bien. Proxy stealth (9 créditos).",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"sku":{"type":"string"}}}}}}`,
	},
}

func main() {
	tenantIDStr := getEnv("TENANT_ID", "")
	if tenantIDStr == "" {
		fmt.Fprintln(os.Stderr, "TENANT_ID env var is required")
		os.Exit(1)
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid TENANT_ID: %v\n", err)
		os.Exit(1)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "webdata"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping DB: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeding %d national sources for tenant %s...\n", len(nationalSources), tenantID)

	inserted, skipped := 0, 0
	for _, s := range nationalSources {
		err := insertSource(db, tenantID, s)
		if err != nil {
			fmt.Printf("  SKIP  %s: %v\n", s.name, err)
			skipped++
			continue
		}
		fmt.Printf("  OK    %s (tier %d, %s)\n", s.name, s.tier, s.priority)
		inserted++
	}

	fmt.Printf("\nDone: %d inserted, %d skipped.\n", inserted, skipped)
}

func insertSource(db *sql.DB, tenantID uuid.UUID, s sourceRow) error {
	query := `
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type,
			city, priority, tier, firecrawl_method, cron_expression,
			notes, extraction_schema, is_active, health_score
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12::jsonb, true, 1.00
		)
		ON CONFLICT (tenant_id, name) DO NOTHING`

	result, err := db.Exec(query,
		tenantID,
		s.name,
		s.baseURL,
		s.category,
		s.sourceType,
		s.city,
		s.priority,
		s.tier,
		s.firecrawlMethod,
		s.cronExpression,
		s.notes,
		s.extractionSchema,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("already exists (skipped)")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
