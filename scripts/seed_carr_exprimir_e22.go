//go:build ignore

// seed_carr_exprimir_e22 seeds the 69 Carrefour sub-category sources that
// are needed to bypass the VTEX 2500-product cap on 6 root categories.
//
// Context: E22 seeded Carrefour by root category (fq=C:/{rootId}/). Six roots
// have >2500 products and VTEX rejects _from>2500 → those roots are capped.
// Solution: scrape each sub-category with the FULL path (fq=C:/{rootId}/{subId}/)
// so each batch stays under 2500. The fq values in carr_exprimir_e22.json were
// pre-validated: all sub-cat counts < 2500.
//
// The source names use "Carrefour - {root} > {sub}" to avoid collision with
// the existing root sources "Carrefour - {root}" seeded in E22.
//
// Usage:
//
//	go run scripts/seed_carr_exprimir_e22.go
//
// Required env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, TENANT_ID
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hornosg/go-shared/infrastructure/postgres"
)

// carrSubSource mirrors the JSON entries from carr_exprimir_e22.json.
type carrSubSource struct {
	root         string
	sub          string
	fq           string
	businessType string
}

// carrExprimir69 contains all 69 Carrefour sub-category paths from
// services/webdata-service/scripts/carr_exprimir_e22.json (validated 2026-06-17).
// fq values are used verbatim in the VTEX search URL.
var carrExprimir69 = []carrSubSource{
	// ── Almacén (root 161) ──────────────────────────────────────────────────────
	{"Almacén", "Aceites y vinagres", "C:/161/162/", "almacen"},
	{"Almacén", "Pastas secas", "C:/161/168/", "almacen"},
	{"Almacén", "Arroz y legumbres", "C:/161/172/", "almacen"},
	{"Almacén", "Harinas", "C:/161/176/", "almacen"},
	{"Almacén", "Enlatados y Conservas", "C:/161/183/", "almacen"},
	{"Almacén", "Sal, aderezos y saborizadores", "C:/161/190/", "almacen"},
	{"Almacén", "Caldos, sopas y puré", "C:/161/195/", "almacen"},
	{"Almacén", "Repostería y postres", "C:/161/199/", "almacen"},
	{"Almacén", "Snacks", "C:/161/214/", "almacen"},
	{"Almacén", "Comidas instantáneas", "C:/161/658/", "almacen"},
	// ── Desayuno y merienda (root 222) ─────────────────────────────────────────
	{"Desayuno y merienda", "Golosinas y chocolates", "C:/222/208/", "almacen"},
	{"Desayuno y merienda", "Galletitas bizcochitos y tostadas", "C:/222/223/", "almacen"},
	{"Desayuno y merienda", "Budines y magdalenas", "C:/222/229/", "almacen"},
	{"Desayuno y merienda", "Yerba", "C:/222/232/", "almacen"},
	{"Desayuno y merienda", "Café", "C:/222/233/", "almacen"},
	{"Desayuno y merienda", "Infusiones", "C:/222/238/", "almacen"},
	{"Desayuno y merienda", "Azúcar y endulzantes", "C:/222/242/", "almacen"},
	{"Desayuno y merienda", "Mermeladas y otros dulces", "C:/222/246/", "almacen"},
	{"Desayuno y merienda", "Cereales y barritas", "C:/222/250/", "almacen"},
	// ── Bebidas (root 255) ──────────────────────────────────────────────────────
	{"Bebidas", "Cervezas", "C:/255/256/", "vinoteca"},
	{"Bebidas", "Vinos", "C:/255/257/", "vinoteca"},
	{"Bebidas", "Fernet y aperitivos", "C:/255/262/", "vinoteca"},
	{"Bebidas", "Bebidas blancas", "C:/255/266/", "vinoteca"},
	{"Bebidas", "Espumantes y sidras", "C:/255/273/", "vinoteca"},
	{"Bebidas", "Gaseosas", "C:/255/277/", "vinoteca"},
	{"Bebidas", "Aguas", "C:/255/283/", "vinoteca"},
	{"Bebidas", "Jugos", "C:/255/286/", "vinoteca"},
	{"Bebidas", "Bebidas energizantes", "C:/255/290/", "vinoteca"},
	{"Bebidas", "Bebidas isotónicas", "C:/255/291/", "vinoteca"},
	{"Bebidas", "Envases", "C:/255/525/", "vinoteca"},
	// ── Lácteos y frescos (root 292) ────────────────────────────────────────────
	{"Lácteos y frescos", "Leches", "C:/292/293/", "fiambreria"},
	{"Lácteos y frescos", "Yogures", "C:/292/299/", "fiambreria"},
	{"Lácteos y frescos", "Mantecas, margarinas y levaduras", "C:/292/302/", "fiambreria"},
	{"Lácteos y frescos", "Cremas de leche", "C:/292/304/", "fiambreria"},
	{"Lácteos y frescos", "Postres", "C:/292/305/", "fiambreria"},
	{"Lácteos y frescos", "Huevos", "C:/292/306/", "fiambreria"},
	{"Lácteos y frescos", "Tapas y pastas frescas", "C:/292/307/", "fiambreria"},
	{"Lácteos y frescos", "Dulce de membrillo y otros dulces", "C:/292/308/", "fiambreria"},
	{"Lácteos y frescos", "Salchichas", "C:/292/309/", "fiambreria"},
	{"Lácteos y frescos", "Quesos", "C:/292/310/", "fiambreria"},
	{"Lácteos y frescos", "Fiambres", "C:/292/318/", "fiambreria"},
	// ── Limpieza (root 359) ─────────────────────────────────────────────────────
	{"Limpieza", "Limpieza de la ropa", "C:/359/360/", "limpieza"},
	{"Limpieza", "Limpieza de pisos y muebles", "C:/359/367/", "limpieza"},
	{"Limpieza", "Insecticidas", "C:/359/376/", "limpieza"},
	{"Limpieza", "Limpieza de cocina", "C:/359/377/", "limpieza"},
	{"Limpieza", "Lavandinas", "C:/359/384/", "limpieza"},
	{"Limpieza", "Rollos de cocina y servilletas", "C:/359/385/", "limpieza"},
	{"Limpieza", "Papeles higiénicos", "C:/359/386/", "limpieza"},
	{"Limpieza", "Limpieza de baño", "C:/359/387/", "limpieza"},
	{"Limpieza", "Desodorantes de ambiente", "C:/359/390/", "limpieza"},
	{"Limpieza", "Artículos de limpieza", "C:/359/394/", "limpieza"},
	// ── Perfumería y farmacia (root 402) ────────────────────────────────────────
	{"Perfumería y farmacia", "Cuidado dental", "C:/402/412/", "farmacia"},
	{"Perfumería y farmacia", "Jabones", "C:/402/418/", "farmacia"},
	{"Perfumería y farmacia", "Protección femenina", "C:/402/422/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado de la piel", "C:/402/427/", "farmacia"},
	{"Perfumería y farmacia", "Antitranspirantes y desodorantes", "C:/402/435/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado corporal", "C:/402/438/", "farmacia"},
	{"Perfumería y farmacia", "Repelentes", "C:/402/443/", "farmacia"},
	{"Perfumería y farmacia", "Algodones e hisopos", "C:/402/444/", "farmacia"},
	{"Perfumería y farmacia", "Farmacia y ortopedia", "C:/402/445/", "farmacia"},
	{"Perfumería y farmacia", "Perfumes y maquillaje", "C:/402/514/", "farmacia"},
	// Cuidado del cabello sub-levels (path has 3 segments: root/mid/leaf)
	{"Perfumería y farmacia", "Cuidado del cabello > Shampoos", "C:/402/403/404/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Acondicionadores", "C:/402/403/405/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Tratamientos capilares", "C:/402/403/406/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Cremas para peinar", "C:/402/403/407/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Coloración", "C:/402/403/408/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Gel y fijadores", "C:/402/403/409/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Piojicidas", "C:/402/403/410/", "farmacia"},
	{"Perfumería y farmacia", "Cuidado del cabello > Accesorios para el cabello", "C:/402/403/411/", "farmacia"},
}

// Carrefour private-label brands — same list as E22 root sources.
var carrExcludedBrands = []string{
	"Carrefour", "Bulnez", "Simpl", "Reder",
	"Carrefour Classic", "Carrefour Selection", "Carrefour Baby",
}

func main() {
	tenantIDStr := getEnvCarr("TENANT_ID", "")
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
		Host:     getEnvCarr("DB_HOST", "localhost"),
		Port:     getEnvCarr("DB_PORT", "5432"),
		User:     getEnvCarr("DB_USER", "postgres"),
		Password: getEnvCarr("DB_PASSWORD", "postgres"),
		DBName:   getEnvCarr("DB_NAME", "webdata_db"),
		SSLMode:  "disable",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Seeding %d Carrefour sub-category sources (E22-exprimir) for tenant %s...\n\n", len(carrExprimir69), tenantID)

	inserted, skipped := 0, 0
	for _, s := range carrExprimir69 {
		name := fmt.Sprintf("Carrefour - %s > %s", s.root, s.sub)
		// Base URL includes the full fq path (VTEX requires the complete path, not just the leaf ID).
		baseURL := fmt.Sprintf(
			"https://www.carrefour.com.ar/api/catalog_system/pub/products/search/?fq=%s",
			s.fq,
		)
		category := fmt.Sprintf("carr_sub_%s", sanitizeCategory(s.root, s.sub))
		notes := fmt.Sprintf(
			"VTEX pública. Sub-cat de E22-exprimir. fq=%s. business_type=%s. Seeded 2026-06-17.",
			s.fq, s.businessType,
		)

		id, err := insertCarrSource(db, tenantID, name, baseURL, category, s.businessType, notes)
		if err != nil {
			fmt.Printf("  SKIP  %s: %v\n", name, err)
			skipped++
			continue
		}
		fmt.Printf("  OK    %s → source_id=%s\n", name, id)
		inserted++
	}

	fmt.Printf("\nDone: %d inserted, %d skipped.\n", inserted, skipped)
	fmt.Println("\nNext: trigger all new sources via the API.")
	fmt.Printf("  TENANT_ID=%s go run scripts/trigger_carr_exprimir_e22.go\n", tenantID)
}

func insertCarrSource(db *sql.DB, tenantID uuid.UUID, name, baseURL, category, businessType, notes string) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type,
			city, priority, tier, firecrawl_method, cron_expression,
			notes, excluded_brands, is_active, health_score
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 'ecommerce',
			'Nacional', 'medium', 2, 'http_json', '0 3 * * 4',
			$5, $6, true, 1.00
		)
		ON CONFLICT (tenant_id, name) DO NOTHING
		RETURNING id`,
		tenantID,
		name,
		baseURL,
		category,
		notes,
		pqArrayCarr(carrExcludedBrands),
	).Scan(&id)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("already exists (skipped)")
	}
	return id, err
}

// sanitizeCategory produces a short stable slug from root+sub for the category column.
func sanitizeCategory(root, sub string) string {
	combined := root + "_" + sub
	result := make([]byte, 0, len(combined))
	for _, c := range combined {
		switch {
		case c >= 'a' && c <= 'z':
			result = append(result, byte(c))
		case c >= 'A' && c <= 'Z':
			result = append(result, byte(c+32)) // toLower
		case c == ' ' || c == ',' || c == '.' || c == '-' || c == '>' || c == '/':
			if len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
		// Skip accented chars and special chars — slug stays ASCII-safe
		}
	}
	// Trim trailing underscore
	for len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}
	// Truncate to 60 chars to be safe with DB limits
	if len(result) > 60 {
		result = result[:60]
	}
	return string(result)
}

func pqArrayCarr(ss []string) interface{} {
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

func getEnvCarr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
