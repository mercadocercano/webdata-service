//go:build ignore

// seed_cordiez_sources seeds all E19 Cordiez VTEX API sources into webdata_sources.
//
// Usage:
//
//	go run scripts/seed_cordiez_sources.go
//
// Required env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, TENANT_ID
//
// Sources seeded (10 categories, all E19):
//   - Cordiez - Almacén           (VTEX category id=10000, business_type=almacen)
//   - Cordiez - Bebidas            (VTEX category id=10001, business_type=vinoteca)   ← E19 cierre: era almacen
//   - Cordiez - Lácteos            (VTEX category id=10003, business_type=fiambreria)
//   - Cordiez - Fiambres y Quesos  (VTEX category id=10006, business_type=fiambreria)
//   - Cordiez - Desayuno y Merienda(VTEX category id=10008, business_type=almacen)
//   - Cordiez - Mascotas           (VTEX category id=10010, business_type=veterinaria) ← E19 cierre: era almacen
//   - Cordiez - Limpieza y Hogar   (VTEX category id=10011, business_type=limpieza)
//   - Cordiez - Cuidado Personal   (VTEX category id=10012, business_type=farmacia)
//   - Cordiez - Kiosco             (VTEX category id=10014, business_type=kiosco)
//   - Cordiez - Cuidado de la Ropa (VTEX category id=10017, business_type=limpieza)
//
// Gate validated 2026-06-17: Lácteos (409 prods) + Limpieza y Hogar (874 prods) OK.
// EAN presente, precio, imagen VTEX CDN, excluded_brands=["Cordiez"] funcional.
//
// After seeding, trigger scrapes via:
//
//	curl -s -X POST http://localhost:8150/api/v1/sources/<source_id>/trigger \
//	  -H "X-API-Key: marketplace-admin-key-2025" \
//	  -H "X-User-Role: marketplace_admin"
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hornosg/go-shared/infrastructure/postgres"
)

type sourceRow struct {
	name            string
	baseURL         string
	category        string
	firecrawlMethod string
	cronExpression  string
	notes           string
	excludedBrands  []string
}

// cordiezSources lists all E19 Cordiez categories.
// Gate validated 2026-06-17: Lácteos (409 prods) + Limpieza y Hogar (874 prods) OK.
// All categories enabled after gate pass.
//
// VTEX category IDs confirmed via /api/catalog_system/pub/category/tree/2:
//   Almacén=10000, Bebidas=10001, Lácteos=10003, Limpieza y Hogar=10011,
//   Fiambres y Quesos=10006, Desayuno y Merienda=10008, Kiosco=10014,
//   Cuidado Personal=10012, Cuidado de la Ropa=10017, Mascotas=10010
var cordiezSources = []sourceRow{
	{
		name:            "Cordiez - Almacén",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10000/",
		category:        "almacen",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 2", // martes 04:00 UTC
		notes:           "API VTEX pública. Categoría Almacén (id=10000). EAN en items[0].ean. private-label Cordiez excluido.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Bebidas",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10001/",
		category:        "bebidas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 2",
		notes:           "API VTEX pública. Categoría Bebidas (id=10001). EAN en items[0].ean. private-label Cordiez excluido. business_type=vinoteca (E19 cierre: era almacen).",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Lácteos",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10003/",
		category:        "lacteos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 2", // martes 05:00 UTC
		notes:           "API VTEX pública. Categoría Lácteos (id=10003). Gate validado 2026-06-17 (409 productos). business_type=fiambreria.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Fiambres y Quesos",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10006/",
		category:        "fiambres_quesos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 2",
		notes:           "API VTEX pública. Categoría Fiambres y Quesos (id=10006). business_type=fiambreria. EAN en items[0].ean.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Desayuno y Merienda",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10008/",
		category:        "desayuno_merienda",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 2",
		notes:           "API VTEX pública. Categoría Desayuno y Merienda (id=10008). business_type=almacen.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Mascotas",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10010/",
		category:        "mascotas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 2",
		notes:           "API VTEX pública. Categoría Mascotas (id=10010). business_type=veterinaria (E19 cierre: era almacen).",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Limpieza y Hogar",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10011/",
		category:        "limpieza_hogar",
		firecrawlMethod: "http_json",
		cronExpression:  "0 6 * * 2", // martes 06:00 UTC (grande: 874 prods)
		notes:           "API VTEX pública. Categoría Limpieza y Hogar (id=10011). Gate validado 2026-06-17 (874 productos). business_type=limpieza.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Cuidado Personal",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10012/",
		category:        "cuidado_personal",
		firecrawlMethod: "http_json",
		cronExpression:  "0 6 * * 2",
		notes:           "API VTEX pública. Categoría Cuidado Personal (id=10012). business_type=farmacia.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Kiosco",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10014/",
		category:        "kiosco",
		firecrawlMethod: "http_json",
		cronExpression:  "0 6 * * 2",
		notes:           "API VTEX pública. Categoría Kiosco (id=10014). business_type=kiosco.",
		excludedBrands:  []string{"Cordiez"},
	},
	{
		name:            "Cordiez - Cuidado de la Ropa",
		baseURL:         "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10017/",
		category:        "cuidado_ropa",
		firecrawlMethod: "http_json",
		cronExpression:  "0 7 * * 2",
		notes:           "API VTEX pública. Categoría Cuidado de la Ropa (id=10017). business_type=limpieza.",
		excludedBrands:  []string{"Cordiez"},
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

	db, err := postgres.Connect(postgres.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "webdata"),
		SSLMode:  "disable",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Seeding %d Cordiez sources for tenant %s...\n", len(cordiezSources), tenantID)

	inserted, skipped := 0, 0
	for _, s := range cordiezSources {
		id, err := insertSource(db, tenantID, s)
		if err != nil {
			fmt.Printf("  SKIP  %s: %v\n", s.name, err)
			skipped++
			continue
		}
		fmt.Printf("  OK    %s → source_id=%s\n", s.name, id)
		inserted++
	}

	fmt.Printf("\nDone: %d inserted, %d skipped.\n", inserted, skipped)
	fmt.Println()
	fmt.Println("To trigger the first scrape for each source:")
	fmt.Println("  curl -s -X POST http://localhost:8150/api/v1/scraping/jobs/trigger \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -H 'Authorization: Bearer <jwt>' \\")
	fmt.Println(`    -H "X-Tenant-ID: ` + tenantID.String() + `" \`)
	fmt.Println(`    -d '{"source_id": "<source_uuid_from_output_above>"}'`)
}

func insertSource(db *sql.DB, tenantID uuid.UUID, s sourceRow) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type,
			city, priority, tier, firecrawl_method, cron_expression,
			notes, excluded_brands, is_active, health_score
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 'ecommerce',
			'Córdoba', 'medium', 2, $5, $6,
			$7, $8, true, 1.00
		)
		ON CONFLICT (tenant_id, name) DO NOTHING
		RETURNING id`,
		tenantID,
		s.name,
		s.baseURL,
		s.category,
		s.firecrawlMethod,
		s.cronExpression,
		s.notes,
		pqArray(s.excludedBrands),
	).Scan(&id)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("already exists (skipped)")
	}
	return id, err
}

// pqArray wraps a []string so lib/pq encodes it as a PostgreSQL text array literal.
func pqArray(ss []string) interface{} {
	// Use the lib/pq array type without importing the package directly
	// to keep this script's imports minimal. Build the literal manually.
	if len(ss) == 0 {
		return "{}"
	}
	// Build: {"val1","val2",...}
	result := "{"
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += `"` + s + `"`
	}
	result += "}"
	return result
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
