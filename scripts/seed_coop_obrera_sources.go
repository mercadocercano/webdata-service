//go:build ignore

// seed_coop_obrera_sources seeds the 6 Cooperativa Obrera (La Coope en Casa) sources into webdata_sources.
//
// API: POST https://api.lacoopeencasa.coop/api/articulos/pagina (no auth required)
// Envelope: {estado:1=OK, datos:{articulos[], cantidad_articulos}}
//
// Usage:
//
//	go run scripts/seed_coop_obrera_sources.go
//
// Required env vars: DB_HOST, DB_PORT, DB_USER, DB_NAME, TENANT_ID
// (DB_PASSWORD defaults to empty — lab-postgres uses trust auth)
//
// Sources seeded (6 root categories):
//   - Coop. Obrera - Almacén       (id=2, category=coop_almacen,    business_type=almacen)
//   - Coop. Obrera - Frescos        (id=3, category=coop_frescos,    business_type=fiambreria)
//   - Coop. Obrera - Bebidas        (id=4, category=coop_bebidas,    business_type=vinoteca)
//   - Coop. Obrera - Perfumería     (id=5, category=coop_perfumeria, business_type=perfumeria)
//   - Coop. Obrera - Limpieza       (id=6, category=coop_limpieza,   business_type=limpieza)
//   - Coop. Obrera - Casa y Jardín  (id=7, category=coop_casa_jardin,business_type=bazar)
//
// CrawlConfig JSON per source sets:
//   - http_method: "POST"
//   - pagination_strategy: "page_increment"
//   - category_id: <root category id>
//   - items_json_path: "datos.articulos"
//
// excluded_brands (E17 decision, memory #96):
//   ["Cooperativa","Ecoop","Sombra de Toro","Coop","La Canasta de la Coope","Primer Precio"]
//
// Gate validation:
//   Run Almacén (id=2) first. Validate RawProducts in webdata_products with
//   precio + imagen_url, then sync to PIM. After gate passes, run other 5 categories.
//
// After seeding, trigger scrapes via:
//
//	curl -s -X POST http://localhost:8170/api/v1/sources/<source_id>/trigger \
//	  -H "X-API-Key: marketplace-admin-key-2025" \
//	  -H "X-User-Role: marketplace_admin"
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// coopCrawlConfig matches the CrawlConfig struct fields needed for Coop. Obrera.
type coopCrawlConfig struct {
	HTTPMethod          string `json:"http_method"`
	RequestBodyTemplate string `json:"request_body_template,omitempty"`
	CategoryID          int    `json:"category_id"`
	ItemsJSONPath       string `json:"items_json_path"`
	PaginationStrategy  string `json:"pagination_strategy"`
}

// coopSource defines one Coop. Obrera category source.
type coopSource struct {
	name           string
	category       string // maps to business_type in category_business_type_map.go
	categoryID     int    // Coop. Obrera root category id (2-7)
	cronExpression string
	notes          string
}

// excludedBrands are Coop. Obrera private-label brands to filter out.
// Decision from E17 (memory #96): exclude own-brand products from the catalog.
var coopExcludedBrands = []string{
	"Cooperativa",
	"Ecoop",
	"Sombra de Toro",
	"Coop",
	"La Canasta de la Coope",
	"Primer Precio",
}

// coopBaseURL is the Coop. Obrera internal JSON API endpoint.
const coopBaseURL = "https://api.lacoopeencasa.coop/api/articulos/pagina"

// coopSources lists all 6 root categories.
// IMPORTANT: category names must match keys in category_business_type_map.go.
// business_type by rubro real (E21 decision — NOT collapsing to almacen):
//   - Almacén → almacen (id=2)
//   - Frescos → fiambreria (lácteos, fiambres, quesos) (id=3)
//   - Bebidas → vinoteca (same as Cordiez Bebidas, E19) (id=4)
//   - Perfumería → perfumeria (Coop labels it as perfumería, not farmacia) (id=5)
//   - Limpieza → limpieza (id=6)
//   - Casa y Jardín → bazar (artículos del hogar) (id=7)
var coopSources = []coopSource{
	{
		name:           "Coop. Obrera - Almacén",
		category:       "coop_almacen",
		categoryID:     2,
		cronExpression: "0 2 * * 3", // miércoles 02:00 UTC
		notes:          "API JSON Coop. Obrera. Categoría Almacén (id=2). ~1896 productos. POST page_increment. Sin EAN, dedup por nombre+marca.",
	},
	{
		name:           "Coop. Obrera - Frescos",
		category:       "coop_frescos",
		categoryID:     3,
		cronExpression: "0 3 * * 3",
		notes:          "API JSON Coop. Obrera. Categoría Frescos (id=3). Lácteos, fiambres, quesos → business_type=fiambreria. POST page_increment.",
	},
	{
		name:           "Coop. Obrera - Bebidas",
		category:       "coop_bebidas",
		categoryID:     4,
		cronExpression: "0 3 * * 3",
		notes:          "API JSON Coop. Obrera. Categoría Bebidas (id=4). business_type=vinoteca (igual que Cordiez Bebidas, E19). POST page_increment.",
	},
	{
		name:           "Coop. Obrera - Perfumería",
		category:       "coop_perfumeria",
		categoryID:     5,
		cronExpression: "0 4 * * 3",
		notes:          "API JSON Coop. Obrera. Categoría Perfumería (id=5). business_type=perfumeria (Coop la llama Perfumería, no Farmacia). POST page_increment.",
	},
	{
		name:           "Coop. Obrera - Limpieza",
		category:       "coop_limpieza",
		categoryID:     6,
		cronExpression: "0 4 * * 3",
		notes:          "API JSON Coop. Obrera. Categoría Limpieza (id=6). business_type=limpieza. POST page_increment.",
	},
	{
		name:           "Coop. Obrera - Casa y Jardín",
		category:       "coop_casa_jardin",
		categoryID:     7,
		cronExpression: "0 5 * * 3",
		notes:          "API JSON Coop. Obrera. Categoría Casa y Jardín (id=7). business_type=bazar. POST page_increment.",
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
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_NAME", "webdata_db"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to DB: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeding %d Coop. Obrera sources for tenant %s...\n", len(coopSources), tenantID)
	fmt.Println()

	inserted, skipped := 0, 0
	for _, s := range coopSources {
		id, err := insertCoopSource(db, tenantID, s)
		if err != nil {
			fmt.Printf("  SKIP  %s: %v\n", s.name, err)
			skipped++
			continue
		}
		fmt.Printf("  OK    %s → source_id=%s (category_id=%d)\n", s.name, id, s.categoryID)
		inserted++
	}

	fmt.Printf("\nDone: %d inserted, %d skipped.\n", inserted, skipped)
	fmt.Println()
	fmt.Println("Gate validation — run Almacén first:")
	fmt.Println("  curl -s -X POST http://localhost:8170/api/v1/scraping/jobs/trigger \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -H 'X-API-Key: marketplace-admin-key-2025' \\")
	fmt.Println("    -H 'X-User-Role: marketplace_admin' \\")
	fmt.Printf("    -H 'X-Tenant-ID: %s' \\\n", tenantID)
	fmt.Println(`    -d '{"source_id": "<almacen_source_id>"}'`)
}

func insertCoopSource(db *sql.DB, tenantID uuid.UUID, s coopSource) (string, error) {
	crawlCfg := coopCrawlConfig{
		HTTPMethod:         "POST",
		CategoryID:         s.categoryID,
		ItemsJSONPath:      "datos.articulos",
		PaginationStrategy: "page_increment",
	}
	crawlCfgJSON, err := json.Marshal(crawlCfg)
	if err != nil {
		return "", fmt.Errorf("marshaling crawl_config: %w", err)
	}

	var id string
	err = db.QueryRow(`
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type,
			city, priority, tier, firecrawl_method, cron_expression,
			notes, excluded_brands, crawl_config, is_active, health_score
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 'ecommerce',
			'Córdoba', 'medium', 2, 'http_json', $5,
			$6, $7, $8, true, 1.00
		)
		ON CONFLICT (tenant_id, name) DO NOTHING
		RETURNING id`,
		tenantID,
		s.name,
		coopBaseURL,
		s.category,
		s.cronExpression,
		s.notes,
		buildPGArray(coopExcludedBrands),
		crawlCfgJSON,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("already exists (skipped)")
	}
	return id, err
}

// buildPGArray constructs a PostgreSQL text array literal: {"val1","val2",...}
func buildPGArray(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
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
