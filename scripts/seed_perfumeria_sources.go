//go:build ignore

// seed_perfumeria_sources seeds perfumería ecommerce sources into webdata_sources.
// Usage: TENANT_ID=<uuid> go run scripts/seed_perfumeria_sources.go
// Env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, TENANT_ID
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hornosg/go-shared/infrastructure/postgres"
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
	prompt           string
	notes            string
	extractionSchema string
}

var defaultSchema = `{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"sku":{"type":"string"}}}}}}`

var perfumeriaSources = []sourceRow{
	// ── Tier 1 — Farmacias y perfumerías grandes (catálogos amplios) ──────
	{
		name:             "Farmacity - Perfumería",
		baseURL:          "https://www.farmacity.com/perfumeria",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "high",
		tier:             1,
		firecrawlMethod:  "extract",
		cronExpression:   "0 6 * * 1", // lunes 06:00
		prompt:           "Extraer todos los productos de perfumería visibles: nombre completo con marca y presentación, precio actual en ARS, URL de imagen del producto, marca, categoría (shampoo, crema, desodorante, maquillaje, perfume, etc).",
		notes:            "Sitio VTEX, buen catálogo de perfumería. Paginar por ?page=N. Incluye dermocosmética, maquillaje y fragancias.",
		extractionSchema: defaultSchema,
	},
	{
		name:             "Simplicity",
		baseURL:          "https://www.simplicity.com.ar/perfumeria",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "high",
		tier:             1,
		firecrawlMethod:  "extract",
		cronExpression:   "0 6 * * 2", // martes 06:00
		prompt:           "Extraer productos de perfumería: nombre completo del producto, precio en pesos argentinos, imagen, marca, categoría. Incluir presentación (ml, g) en el nombre.",
		notes:            "Cadena de perfumerías argentina, catálogo completo online. Fuerte en maquillaje y cuidado capilar.",
		extractionSchema: defaultSchema,
	},
	{
		name:             "Pigmento",
		baseURL:          "https://www.pigmento.com.ar/productos",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "high",
		tier:             1,
		firecrawlMethod:  "extract",
		cronExpression:   "0 6 * * 3", // miércoles 06:00
		prompt:           "Extraer productos de cosmética y perfumería: nombre, precio ARS, imagen, marca, categoría. Foco en maquillaje profesional y cuidado personal.",
		notes:            "Perfumería online argentina. Buen surtido de maquillaje (Maybelline, Revlon, L'Oréal) y cuidado facial.",
		extractionSchema: defaultSchema,
	},
	// ── Tier 2 — Distribuidoras y mayoristas ────────────────────────────
	{
		name:             "Distribuidora Pop - Perfumería",
		baseURL:          "https://www.distribuidorapop.com.ar/producto-categoria/perfumeria/",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "medium",
		tier:             2,
		firecrawlMethod:  "extract",
		cronExpression:   "0 6 * * 4", // jueves 06:00
		prompt:           "Extraer productos de perfumería mayorista: nombre completo, precio mayorista en ARS, imagen, marca. Categorías: shampoo, desodorantes, jabones, cremas, pañales.",
		notes:            "Distribuidora mayorista con precios de referencia. Marcas masivas: Dove, Rexona, Sedal, Pantene, Nivea. Descuento por volumen.",
		extractionSchema: defaultSchema,
	},
	{
		name:             "Carlos Benassi - Perfumería",
		baseURL:          "https://benassi.com.ar/16-perfumeria",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Litoral",
		priority:         "medium",
		tier:             2,
		firecrawlMethod:  "extract",
		cronExpression:   "0 6 * * 5", // viernes 06:00
		prompt:           "Extraer productos de perfumería: nombre, precio en ARS, imagen, marca, categoría. Incluye desodorantes, shampoo, cremas faciales y capilares.",
		notes:            "Distribuidor mayorista del Litoral. Buen surtido de marcas líderes (Gillette, Nivea, Pantene, Sedal, TRESemmé).",
		extractionSchema: defaultSchema,
	},
	{
		name:             "Juleriaque",
		baseURL:          "https://www.juleriaque.com.ar/productos",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "medium",
		tier:             2,
		firecrawlMethod:  "extract",
		cronExpression:   "0 7 * * 1", // lunes 07:00
		prompt:           "Extraer productos de perfumería selectiva: nombre completo, precio ARS, imagen, marca. Foco en fragancias, maquillaje premium y dermocosmética.",
		notes:            "Cadena de perfumerías premium. Fragancias importadas, maquillaje selectivo, dermocosmética (La Roche-Posay, Eucerin, Neutrogena).",
		extractionSchema: defaultSchema,
	},
	// ── Tier 2 — Farmacias online (sección belleza) ─────────────────────
	{
		name:             "Farmaplus - Belleza",
		baseURL:          "https://www.farmaplus.com.ar/maquillajes-y-estetica",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "medium",
		tier:             2,
		firecrawlMethod:  "extract",
		cronExpression:   "0 7 * * 2", // martes 07:00
		prompt:           "Extraer productos de maquillaje y estética: nombre, precio ARS, imagen, marca. Categorías: base, labial, máscara, sombras, delineador.",
		notes:            "Sección de belleza de Farmaplus. 95+ productos Maybelline, 55+ Revlon. Buenas imágenes de producto.",
		extractionSchema: defaultSchema,
	},
	{
		name:             "Farmacia Leloir - Perfumería",
		baseURL:          "https://www.farmacialeloir.com.ar/perfumeria",
		category:         "perfumeria",
		sourceType:       "ecommerce",
		city:             "Nacional",
		priority:         "low",
		tier:             2,
		firecrawlMethod:  "extract",
		cronExpression:   "0 7 * * 3", // miércoles 07:00
		prompt:           "Extraer productos de perfumería: nombre completo con marca y presentación, precio en ARS, URL de imagen, marca, categoría.",
		notes:            "Farmacia online con sección perfumería. Complementa con marcas de dermocosmética.",
		extractionSchema: defaultSchema,
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

	fmt.Printf("Seeding %d perfumería sources for tenant %s...\n", len(perfumeriaSources), tenantID)

	inserted, skipped := 0, 0
	for _, s := range perfumeriaSources {
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
			prompt, notes, extraction_schema, is_active, health_score
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13::jsonb, true, 1.00
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
		s.prompt,
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
