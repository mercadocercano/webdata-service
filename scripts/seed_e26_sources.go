//go:build ignore

// seed_e26_sources seeds the E26 VTEX non-food sources into webdata_sources:
// Easy (hogar/construcción → ferretería + electricidad + bazar + piletas + electrodomésticos),
// Blaisten (baños/pisos/griferías → ferretería) y Puppis (mascotas → veterinaria).
//
// Rubros del piloto cubiertos: ferretería (núcleo). SBS y Frávega quedaron FUERA de E26
// por decisión del owner (volumen 124k/321k, revisar más adelante). Kiosco NO entra acá:
// requiere carve-out de las fuentes padre de los supers (conflicto de dedup por EAN, E25).
//
// Usage:
//
//	DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres \
//	  DB_NAME=webdata_db TENANT_ID=<uuid> go run scripts/seed_e26_sources.go
//
// Reusa el mismo adapter VTEX http_json (range/GET) de E19/E22. Sin código nuevo.
//
// Category IDs validados via /api/catalog_system/pub/category/tree/2 (2026-06-17):
//   Easy: Baños/Cocinas=2, Electrodomésticos=3, Muebles=4, Textil/Bazar/Deco=6,
//         Iluminación=7, Jardín y Aire Libre=8 (particionado), Pisos=10, Pinturas=11,
//         Aberturas=12, Construcción=13, Plomería=14, Electricidad=15, Herramientas=16,
//         Ferretería=506. (Jardín hijos: 52,53,55,56,58,59,60,565,571)
//   Blaisten: catálogo completo (773 prods, bajo el cap 2500 → 1 source sin fq).
//   Puppis:   catálogo completo (1912 prods, bajo el cap 2500 → 1 source sin fq).
//
// GOTCHA cap 2500: las cats Jardín(8)=4137 se particionan por subcategoría. Bazar/Deco(6)=2627
// y Ferretería(506)=2570 superan apenas el cap (pierden ~100 de la cola) → aceptable en v1,
// particionar luego si hace falta.
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

const easyBase = "https://www.easy.com.ar/api/catalog_system/pub/products/search/?fq=C:/"

func easySrc(name, catID, category, note string) sourceRow {
	return sourceRow{
		name:            "Easy - " + name,
		baseURL:         easyBase + catID + "/",
		category:        category,
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 1", // lunes 03:00 UTC (escalonado, fuera de los supers)
		notes:           "VTEX pública. " + note,
		excludedBrands:  []string{}, // sin MDD evidente en muestra; validar y ampliar
	}
}

// ─── Easy ────────────────────────────────────────────────────────────────────
var easySources = []sourceRow{
	// Ferretería (rubro núcleo del piloto)
	easySrc("Ferretería", "506", "easy_ferreteria", "Cat Ferretería id=506 (~2570). business_type=ferreteria."),
	easySrc("Herramientas", "16", "easy_herramientas", "Cat Herramientas id=16 (~2196). business_type=ferreteria."),
	easySrc("Construcción y Maderas", "13", "easy_construccion", "Cat Construcción y Maderas id=13 (~1862). business_type=ferreteria."),
	easySrc("Plomería", "14", "easy_plomeria", "Cat Plomería id=14 (~1912). business_type=ferreteria."),
	easySrc("Aberturas", "12", "easy_aberturas", "Cat Aberturas id=12 (~381). business_type=ferreteria."),
	easySrc("Pisos y Revestimientos", "10", "easy_pisos", "Cat Pisos y Revestimientos id=10 (~573). business_type=ferreteria."),
	easySrc("Pinturas", "11", "easy_pinturas", "Cat Pinturas id=11 (~1969). business_type=ferreteria."),
	easySrc("Baños y Cocinas", "2", "easy_banos_cocinas", "Cat Baños y Cocinas id=2 (~1085). business_type=ferreteria."),
	// Electricidad
	easySrc("Electricidad", "15", "easy_electricidad", "Cat Electricidad id=15 (~1388). business_type=electricidad."),
	easySrc("Iluminación", "7", "easy_iluminacion", "Cat Iluminación id=7 (~1281). business_type=electricidad."),
	// Bazar
	easySrc("Textil, Bazar y Deco", "6", "easy_bazar_deco", "Cat Textil/Bazar/Deco id=6 (~2627, supera cap). business_type=bazar."),
	easySrc("Muebles", "4", "easy_muebles", "Cat Muebles id=4 (~1164). business_type=bazar."),
	// Electrodomésticos
	easySrc("Electrodomésticos", "3", "easy_electrodomesticos", "Cat Electrodomésticos id=3 (~684). business_type=electrodomesticos."),
	// Jardín y Aire Libre → piletas (particionado por subcategoría, cat 8=4137 supera cap)
	easySrc("Jardín", "52", "easy_jardin", "Cat Jardín id=52 (hijo de Jardín y Aire Libre). business_type=piletas."),
	easySrc("Muebles de Exterior", "53", "easy_jardin_muebles_ext", "Cat Muebles de exterior id=53. business_type=piletas."),
	easySrc("Parrillas y Accesorios", "55", "easy_jardin_parrillas", "Cat Parrillas y accesorios id=55. business_type=piletas."),
	easySrc("Piletas", "56", "easy_jardin_piletas", "Cat Piletas id=56. business_type=piletas."),
	easySrc("Tiempo Libre", "58", "easy_jardin_tiempolibre", "Cat Tiempo libre id=58. business_type=piletas."),
	easySrc("Camping y Outdoor", "59", "easy_jardin_camping", "Cat Camping y Outdoor id=59. business_type=piletas."),
	easySrc("Herramientas de Jardín", "565", "easy_jardin_herramientas", "Cat Herramientas y Maquinarias de Jardín id=565. business_type=ferreteria."),
	easySrc("Armados de Jardín", "571", "easy_jardin_armados", "Cat Armados de Jardín id=571. business_type=piletas."),
	easySrc("Mascotas (Jardín)", "60", "easy_jardin_mascotas", "Cat Mascotas id=60 (bajo Jardín). business_type=veterinaria."),
}

// ─── Blaisten ────────────────────────────────────────────────────────────────
// Base: https://www.blaisten.com.ar — 773 prods totales, bajo el cap → 1 source.
var blaistenSources = []sourceRow{
	{
		name:            "Blaisten - Catálogo",
		baseURL:         "https://www.blaisten.com.ar/api/catalog_system/pub/products/search/?",
		category:        "blaisten_general",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 1",
		notes:           "VTEX pública. Catálogo completo (~773 prods, baños/pisos/griferías/sanitarios). business_type=ferreteria. Validado 2026-06-17.",
		excludedBrands:  []string{},
	},
}

// ─── Puppis ──────────────────────────────────────────────────────────────────
// Base: https://www.puppis.com.ar — 1912 prods totales, bajo el cap → 1 source.
var puppisSources = []sourceRow{
	{
		name:            "Puppis - Catálogo",
		baseURL:         "https://www.puppis.com.ar/api/catalog_system/pub/products/search/?",
		category:        "puppis_general",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 1",
		notes:           "VTEX pública. Catálogo completo (~1912 prods, alimento/accesorios mascotas). business_type=veterinaria. Validado 2026-06-17.",
		excludedBrands:  []string{},
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
		DBName:   getEnv("DB_NAME", "webdata_db"),
		SSLMode:  "disable",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	allSources := []struct {
		label   string
		sources []sourceRow
	}{
		{"Easy", easySources},
		{"Blaisten", blaistenSources},
		{"Puppis", puppisSources},
	}

	total := 0
	for _, g := range allSources {
		total += len(g.sources)
	}
	fmt.Printf("Seeding %d E26 VTEX no-comida sources for tenant %s...\n\n", total, tenantID)

	inserted, skipped := 0, 0
	for _, g := range allSources {
		fmt.Printf("── %s ──\n", g.label)
		for _, s := range g.sources {
			id, err := insertSource(db, tenantID, s)
			if err != nil {
				fmt.Printf("  SKIP  %s: %v\n", s.name, err)
				skipped++
				continue
			}
			fmt.Printf("  OK    %s → source_id=%s\n", s.name, id)
			inserted++
		}
		fmt.Println()
	}

	fmt.Printf("Done: %d inserted, %d skipped.\n", inserted, skipped)
}

func insertSource(db *sql.DB, tenantID uuid.UUID, s sourceRow) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type,
			city, priority, tier, firecrawl_method, cron_expression,
			notes, excluded_brands, is_active, health_score, authoritative_category
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 'ecommerce',
			'Nacional', 'medium', 2, $5, $6,
			$7, $8, true, 1.00, true
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

func pqArray(ss []string) interface{} {
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
