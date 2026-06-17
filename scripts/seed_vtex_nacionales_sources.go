//go:build ignore

// seed_vtex_nacionales_sources seeds all E22 VTEX national supermarket sources
// into webdata_sources: Día, Carrefour, La Anónima (MasOnline), Farmacity.
//
// Usage:
//
//	go run scripts/seed_vtex_nacionales_sources.go
//
// Required env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, TENANT_ID
//
// These sources use the same VTEX http_json adapter as Cordiez (E19).
// The adapter handles the "Resources" header (used by Día/Carrefour/MasOnline/Farmacity)
// in addition to "Content-Range" (used by Cordiez). No code change needed in the adapter.
//
// Category IDs validated via /api/catalog_system/pub/category/tree/1 on 2026-06-17:
//   Día:       Almacén=1, Desayuno=80, Frescos=121, Bebidas=164, Congelados=200,
//              Perfumería=216, Limpieza=282, Mascotas=71
//   Carrefour: Almacén=161, Desayuno y merienda=222, Bebidas=255, Lácteos y frescos=292,
//              Panadería=336, Congelados=347, Limpieza=359, Perfumería y farmacia=402,
//              Mascotas=471
//   MasOnline: Aceites/Aderezos=200005, Arroz/Pastas=200009, Desayunos=200039,
//              Lácteos=200066, Fiambres=200046, Congelados=200028, Gaseosas=200051,
//              Limpieza=200086, Farmacia=200092
//   Farmacity: Farmacia=199, Medicamentos Venta Libre=979, Cuidado Personal=92,
//              Cuidado de la Piel=93, Cuidado Capilar=130, Perfumes=80,
//              Maquillaje=2, Hogar y Alimentos=301
//
// excluded_brands validated with live sample 2026-06-17. Ampliar si aparecen más MDD.
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

// ─── Día ─────────────────────────────────────────────────────────────────────
// Base: https://diaonline.supermercadosdia.com.ar
// Category IDs confirmed via /api/catalog_system/pub/category/tree/1 (2026-06-17)
// API returns HTTP 206 + Resources header (same VTEX contract as Cordiez).
var diaSources = []sourceRow{
	{
		name:            "Día - Almacén",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/1/",
		category:        "dia_almacen",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 3", // miércoles 03:00 UTC
		notes:           "VTEX pública. Cat Almacén id=1. Validado 2026-06-17 (1138 prods). Excluded: DIA/Día private-label.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Desayuno",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/80/",
		category:        "dia_desayuno",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 3",
		notes:           "VTEX pública. Cat Desayuno id=80. business_type=almacen.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Frescos",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/121/",
		category:        "dia_frescos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 3",
		notes:           "VTEX pública. Cat Frescos id=121. business_type=fiambreria (lácteos, fiambres, quesos).",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Bebidas",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/164/",
		category:        "dia_bebidas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 3",
		notes:           "VTEX pública. Cat Bebidas id=164. business_type=vinoteca (igual convención E19 Cordiez Bebidas).",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Congelados",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/200/",
		category:        "dia_congelados",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 3",
		notes:           "VTEX pública. Cat Congelados id=200. business_type=almacen.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Perfumería",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/216/",
		category:        "dia_perfumeria",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 3",
		notes:           "VTEX pública. Cat Perfumería id=216. business_type=perfumeria.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Limpieza",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/282/",
		category:        "dia_limpieza",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 3",
		notes:           "VTEX pública. Cat Limpieza id=282. business_type=limpieza.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
	{
		name:            "Día - Mascotas",
		baseURL:         "https://diaonline.supermercadosdia.com.ar/api/catalog_system/pub/products/search/?fq=C:/71/",
		category:        "dia_mascotas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 3",
		notes:           "VTEX pública. Cat Mascotas id=71. business_type=veterinaria.",
		excludedBrands:  []string{"DIA", "Dia", "Bonté", "As de Oros", "Delicious"},
	},
}

// ─── Carrefour ───────────────────────────────────────────────────────────────
// Base: https://www.carrefour.com.ar
// Category IDs confirmed via /api/catalog_system/pub/category/tree/1 (2026-06-17)
// Sample de 50 prods: "Genérico" apareció como marca → incluido en excluded_brands.
var carrefourSources = []sourceRow{
	{
		name:            "Carrefour - Almacén",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/161/",
		category:        "carr_almacen",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 4", // jueves 03:00 UTC
		notes:           "VTEX pública. Cat Almacén id=161. Validado 2026-06-17 (6101 prods). Excluded: marcas Carrefour + Genérico.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Desayuno y Merienda",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/222/",
		category:        "carr_desayuno",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 4",
		notes:           "VTEX pública. Cat Desayuno y merienda id=222. business_type=almacen.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Bebidas",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/255/",
		category:        "carr_bebidas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 4",
		notes:           "VTEX pública. Cat Bebidas id=255. business_type=vinoteca.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Lácteos y Frescos",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/292/",
		category:        "carr_lacteos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 4",
		notes:           "VTEX pública. Cat Lácteos y productos frescos id=292. business_type=fiambreria (igual que Cordiez Lácteos E19).",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Panadería",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/336/",
		category:        "carr_panaderia",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 4",
		notes:           "VTEX pública. Cat Panadería id=336. business_type=almacen.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Congelados",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/347/",
		category:        "carr_congelados",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 4",
		notes:           "VTEX pública. Cat Congelados id=347. business_type=almacen.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Limpieza",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/359/",
		category:        "carr_limpieza",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 4",
		notes:           "VTEX pública. Cat Limpieza id=359. business_type=limpieza.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Perfumería y Farmacia",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/402/",
		category:        "carr_farmacia",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 4",
		notes:           "VTEX pública. Cat Perfumería y farmacia id=402. business_type=farmacia (cubre ambos rubros).",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
	{
		name:            "Carrefour - Mascotas",
		baseURL:         "https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=C:/471/",
		category:        "carr_mascotas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 6 * * 4",
		notes:           "VTEX pública. Cat Mascotas id=471. business_type=veterinaria.",
		excludedBrands:  []string{"Carrefour", "Bulnez", "Simpl", "Reder", "Carrefour Classic", "Carrefour Selection", "Carrefour Baby", "Genérico"},
	},
}

// ─── La Anónima / MasOnline ──────────────────────────────────────────────────
// Base: https://www.masonline.com.ar
// Granular (117 cats nivel-1). Objetivo: rubros comida/limpieza/farmacia.
// Category IDs confirmed via /api/catalog_system/pub/category/tree/1 (2026-06-17).
var anonimaSources = []sourceRow{
	{
		name:            "La Anónima - Aceites y Aderezos",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200005/",
		category:        "anon_aceites_aderezos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 5", // viernes 03:00 UTC
		notes:           "VTEX pública. Cat Aceites, Vinagres y Aderezos id=200005. Validado 2026-06-17 (516 prods). business_type=almacen.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Arroz y Pastas",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200009/",
		category:        "anon_arroz_pastas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 5",
		notes:           "VTEX pública. Cat Arroz, Legumbres y Pastas id=200009. business_type=almacen.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Desayunos y Meriendas",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200039/",
		category:        "anon_desayunos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 5",
		notes:           "VTEX pública. Cat Desayunos y Meriendas id=200039. business_type=almacen.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Lácteos",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200066/",
		category:        "anon_lacteos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 5",
		notes:           "VTEX pública. Cat Lácteos id=200066. business_type=fiambreria (igual convención E19).",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Fiambres y Embutidos",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200046/",
		category:        "anon_fiambres",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 5",
		notes:           "VTEX pública. Cat Fiambres y Embutidos id=200046. business_type=fiambreria.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Congelados",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200028/",
		category:        "anon_congelados",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 5",
		notes:           "VTEX pública. Cat Congelados id=200028. business_type=almacen.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Gaseosas",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200051/",
		category:        "anon_gaseosas",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 5",
		notes:           "VTEX pública. Cat Gaseosas id=200051. business_type=vinoteca (bebidas, igual convención E19).",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Limpieza del Hogar",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200086/",
		category:        "anon_limpieza",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 5",
		notes:           "VTEX pública. Cat Limpieza del Hogar id=200086. business_type=limpieza.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
	{
		name:            "La Anónima - Farmacia",
		baseURL:         "https://www.masonline.com.ar/api/catalog_system/pub/products/search/?fq=C:/200092/",
		category:        "anon_farmacia",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 5",
		notes:           "VTEX pública. Cat Farmacia id=200092. Validado 2026-06-17 OK. business_type=farmacia.",
		excludedBrands:  []string{"La Anónima", "Anónima", "Best", "Sucesos", "Nuestras Marcas"},
	},
}

// ─── Farmacity ───────────────────────────────────────────────────────────────
// Base: https://www.farmacity.com
// Category IDs confirmed via /api/catalog_system/pub/category/tree/1 (2026-06-17)
// Sample de 50 prods cat 199: "Farmacity" private-label presente → excluido.
var farmacitySources = []sourceRow{
	{
		name:            "Farmacity - Farmacia",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/199/",
		category:        "farm_farmacia",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 6", // sábado 03:00 UTC
		notes:           "VTEX pública. Cat Farmacia id=199. Validado 2026-06-17 (425 prods). business_type=farmacia.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Medicamentos Venta Libre",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/979/",
		category:        "farm_medicamentos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 3 * * 6",
		notes:           "VTEX pública. Cat Medicamentos Venta Libre id=979. business_type=farmacia.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Cuidado Personal",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/92/",
		category:        "farm_cuidado_personal",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 6",
		notes:           "VTEX pública. Cat Cuidado Personal id=92. business_type=farmacia.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Cuidado de la Piel",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/93/",
		category:        "farm_cuidado_piel",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 6",
		notes:           "VTEX pública. Cat Cuidado de la Piel id=93. business_type=perfumeria.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Cuidado Capilar",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/130/",
		category:        "farm_cuidado_capilar",
		firecrawlMethod: "http_json",
		cronExpression:  "0 4 * * 6",
		notes:           "VTEX pública. Cat Cuidado Capilar id=130. business_type=perfumeria.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Perfumes y Fragancias",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/80/",
		category:        "farm_perfumes",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 6",
		notes:           "VTEX pública. Cat Perfumes y Fragancias id=80. business_type=perfumeria.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Maquillaje",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/2/",
		category:        "farm_maquillaje",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 6",
		notes:           "VTEX pública. Cat Maquillaje id=2. business_type=perfumeria.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
	},
	{
		name:            "Farmacity - Hogar y Alimentos",
		baseURL:         "https://www.farmacity.com/api/catalog_system/pub/products/search/?fq=C:/301/",
		category:        "farm_hogar_alimentos",
		firecrawlMethod: "http_json",
		cronExpression:  "0 5 * * 6",
		notes:           "VTEX pública. Cat Hogar y Alimentos id=301. business_type=almacen.",
		excludedBrands:  []string{"Farmacity", "The Goalkeeper", "Simplicity"},
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

	allSources := []struct {
		label   string
		sources []sourceRow
	}{
		{"Día", diaSources},
		{"Carrefour", carrefourSources},
		{"La Anónima (MasOnline)", anonimaSources},
		{"Farmacity", farmacitySources},
	}

	total := 0
	for _, g := range allSources {
		total += len(g.sources)
	}
	fmt.Printf("Seeding %d E22 VTEX-nacional sources for tenant %s...\n\n", total, tenantID)

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

	fmt.Printf("Done: %d inserted, %d skipped.\n\n", inserted, skipped)
	fmt.Println("To trigger the E2E gate (1 category per source) run:")
	fmt.Printf("  DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=webdata_db TENANT_ID=%s go run scripts/trigger_e22_gate.go\n", tenantID)
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
			'Nacional', 'medium', 2, $5, $6,
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
