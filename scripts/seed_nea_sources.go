//go:build ignore

// seed_nea_sources seeds 15 NEA ecommerce/wholesale sources into webdata_sources.
// Usage: go run scripts/seed_nea_sources.go
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

var neaSources = []sourceRow{
	// ── Tier 1 — Alta prioridad, scraping inmediato ─────────────────────────
	{
		name:            "Maxiconsumo",
		baseURL:         "https://www.maxiconsumo.com",
		category:        "supermercado_mayorista",
		sourceType:      "mayorista",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 1", // lunes 06:00
		notes:           "Catálogo semanal online. Presencia confirmada en NEA. Mercadería del día a día.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"},"sku":{"type":"string"}}}}}}`,
	},
	{
		name:            "Diarco",
		baseURL:         "https://www.diarco.com.ar",
		category:        "supermercado_mayorista",
		sourceType:      "mayorista",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 1", // lunes 06:00
		notes:           "Precios exclusivos con DNI. Sucursal confirmada en NEA (Uruguay 1163).",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"}}}}}}`,
	},
	{
		name:            "Carrefour",
		baseURL:         "https://www.carrefour.com.ar",
		category:        "supermercado",
		sourceType:      "ecommerce",
		city:            "Posadas/Corrientes/Resistencia",
		priority:        "high",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 7 * * *", // diario 07:00
		notes:           "Cadena nacional con ecommerce activo y presencia física en NEA.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"originalPrice":{"type":"number"}}}}}}`,
	},
	{
		name:            "Electro Misiones",
		baseURL:         "https://www.electromisiones.com.ar",
		category:        "electronica",
		sourceType:      "ecommerce_regional",
		city:            "Posadas/NEA",
		priority:        "high",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 1", // lunes 08:00
		notes:           "Principal cadena regional de electrónica del NEA. Sucursales en Posadas, Garupá, Santo Tomé, Ituzaingó.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"},"model":{"type":"string"}}}}}}`,
	},
	{
		name:            "Farmacity",
		baseURL:         "https://www.farmacity.com",
		category:        "farmacia_perfumeria",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            1,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 1", // lunes 08:00
		notes:           "Cadena nacional con ecommerce activo. Medicamentos, salud, belleza.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"laboratoire":{"type":"string"},"activeIngredient":{"type":"string"}}}}}}`,
	},
	// ── Tier 2 — Alta/Media prioridad, scraping semanal ────────────────────
	{
		name:            "Vital",
		baseURL:         "https://www.vital.com.ar",
		category:        "supermercado_mayorista",
		sourceType:      "mayorista",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 6 * * 2", // martes 06:00
		notes:           "Integrado en Precios Justos Barriales. Catálogo actualizado online.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"}}}}}}`,
	},
	{
		name:            "Coto",
		baseURL:         "https://www.coto.com.ar",
		category:        "supermercado",
		sourceType:      "ecommerce",
		city:            "Posadas/Corrientes",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 7 * * 3", // miércoles 07:00
		notes:           "Cadena nacional con buscador de sucursales y compra online.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}`,
	},
	{
		name:            "La Anónima",
		baseURL:         "https://www.laanonima.com.ar",
		category:        "supermercado",
		sourceType:      "ecommerce",
		city:            "Resistencia/NEA",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 7 * * 3", // miércoles 07:00
		notes:           "Fuerte presencia en NEA e interior. Catálogo online activo.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}`,
	},
	{
		name:            "Cetrogar",
		baseURL:         "https://www.cetrogar.com.ar",
		category:        "electronica",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 2", // martes 08:00
		notes:           "ecommerce nacional con fuerte presencia en interior del país.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}`,
	},
	{
		name:            "Easy",
		baseURL:         "https://www.easy.com.ar",
		category:        "construccion_ferreteria",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 9 * * 1", // lunes 09:00
		notes:           "Cadena con sucursales y ecommerce. Catálogo de herramientas, construcción y hogar.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}`,
	},
	{
		name:            "Sodimac",
		baseURL:         "https://www.sodimac.com.ar",
		category:        "construccion_hogar",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 9 * * 1", // lunes 09:00
		notes:           "Catálogo amplio de hogar y construcción. Retiro en tienda y entrega.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"sku":{"type":"string"}}}}}}`,
	},
	{
		name:            "Farmalife",
		baseURL:         "https://www.farmalife.com.ar",
		category:        "farmacia_perfumeria",
		sourceType:      "ecommerce_regional",
		city:            "Corrientes",
		priority:        "high",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 2", // martes 08:00
		notes:           "Sucursal física en Corrientes (Pellegrini 645). Dermocosmética, cuidado personal.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}`,
	},
	{
		name:            "Naldo",
		baseURL:         "https://www.naldo.com.ar",
		category:        "electronica",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "medium",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 4", // jueves 08:00
		notes:           "Envíos a todo el país. Buena referencia de precios de electrónica.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}`,
	},
	{
		name:            "Megatone",
		baseURL:         "https://www.megatone.net",
		category:        "electronica",
		sourceType:      "ecommerce",
		city:            "Nacional/NEA",
		priority:        "medium",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 4", // jueves 08:00
		notes:           "Cuotas sin interés. Envíos nacionales. Referencia de precios electrodomésticos.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}`,
	},
	{
		name:            "Farmar",
		baseURL:         "https://www.farmar.com.ar",
		category:        "farmacia_perfumeria",
		sourceType:      "ecommerce",
		city:            "Nacional",
		priority:        "medium",
		tier:            2,
		firecrawlMethod: "extract",
		cronExpression:  "0 8 * * 3", // miércoles 08:00
		notes:           "Descuentos 2x1, 70-80%. Referencia de precios farmacia online.",
		extractionSchema: `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"discount":{"type":"string"}}}}}}`,
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

	fmt.Printf("Seeding %d NEA sources for tenant %s...\n", len(neaSources), tenantID)

	inserted, skipped := 0, 0
	for _, s := range neaSources {
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
