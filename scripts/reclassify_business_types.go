//go:build ignore

// reclassify_business_types re-clasifica el business_type del catálogo global ya
// sincronizado (pim_db.global_products) reutilizando el resolver de webdata
// (única fuente de verdad: value_object.ResolveBusinessTypeFromProductCategory).
//
// Alcance (decisión owner): "vacíos + correcciones seguras".
//   - RELLENO:    business_type vacío/null  → resolver(category) si resuelve.
//   - CORRECCIÓN: business_type == 'almacen' y resolver(category) != 'almacen' → mover.
//   - SKIP:       ya en un rubro específico (≠ almacen, ≠ vacío); o resolver no resuelve;
//                 o resuelve a lo mismo; o colisión UNIQUE(name, business_type).
//
// DRY-RUN por defecto. Aplica con --apply.
//
// Conecta a PIM (no a webdata): pasar DB_NAME=pim_db (y DB_HOST/PORT/USER/PASSWORD).
//
// Uso (dentro del container webdata, que tiene red al lab-postgres):
//
//	docker exec -e DB_HOST=lab-postgres -e DB_NAME=pim_db -e DB_USER=postgres \
//	  -e DB_PASSWORD=<pass> mc-webdata-service \
//	  sh -c 'cd /app && go run ./scripts/reclassify_business_types.go'          # dry-run
//	  ... go run ./scripts/reclassify_business_types.go --apply                 # apply
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/hornosg/go-shared/infrastructure/postgres"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type change struct {
	id      string
	name    string
	from    string // "" = vacío
	to      string
	kind    string // "relleno" | "correccion"
}

func main() {
	apply := flag.Bool("apply", false, "aplica los cambios (por defecto: dry-run)")
	flag.Parse()

	db, err := postgres.Connect(postgres.Config{
		Host:     getEnv("DB_HOST", "lab-postgres"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "pim_db"),
		SSLMode:  "disable",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	mode := "DRY-RUN (no escribe)"
	if *apply {
		mode = "APPLY (escribe)"
	}
	fmt.Printf("Re-clasificación de business_type — modo: %s\n", mode)
	fmt.Printf("DB: %s@%s/%s\n\n", getEnv("DB_USER", "postgres"), getEnv("DB_HOST", "lab-postgres"), getEnv("DB_NAME", "pim_db"))

	rows, err := db.Query(`SELECT id, name, COALESCE(category,''), COALESCE(business_type,'')
		FROM global_products WHERE source LIKE 'scraper%'`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var changes []change
	var (
		total              int
		skipNoResuelve     int
		skipYaEspecifico   int
		skipMismo          int
	)
	for rows.Next() {
		var id, name, category, current string
		if err := rows.Scan(&id, &name, &category, &current); err != nil {
			fmt.Fprintf(os.Stderr, "scan failed: %v\n", err)
			os.Exit(1)
		}
		total++

		assignment, ok := value_object.ResolveBusinessTypeFromProductCategory(category)
		if !ok {
			skipNoResuelve++
			continue
		}
		resolved := assignment.BusinessTypeCode

		switch {
		case current == "":
			changes = append(changes, change{id, name, "", resolved, "relleno"})
		case current == "almacen" && resolved != "almacen":
			changes = append(changes, change{id, name, "almacen", resolved, "correccion"})
		case current == resolved:
			skipMismo++
		default:
			// ya en un rubro específico distinto → no tocar
			skipYaEspecifico++
		}
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "rows error: %v\n", err)
		os.Exit(1)
	}

	// Detectar colisiones UNIQUE(name, business_type) y, si --apply, ejecutar.
	var (
		applied    int
		collisions []change
	)
	byTarget := map[string][2]int{} // to → [rellenos, correcciones]
	for _, c := range changes {
		var exists bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM global_products WHERE name=$1 AND business_type=$2 AND id<>$3)`,
			c.name, c.to, c.id,
		).Scan(&exists); err != nil {
			fmt.Fprintf(os.Stderr, "collision check failed for %s: %v\n", c.id, err)
			os.Exit(1)
		}
		if exists {
			collisions = append(collisions, c)
			continue
		}

		agg := byTarget[c.to]
		if c.kind == "relleno" {
			agg[0]++
		} else {
			agg[1]++
		}
		byTarget[c.to] = agg

		if *apply {
			if _, err := db.Exec(
				`UPDATE global_products SET business_type=$1, updated_at=now() WHERE id=$2`,
				c.to, c.id,
			); err != nil {
				fmt.Fprintf(os.Stderr, "update failed for %s: %v\n", c.id, err)
				os.Exit(1)
			}
			applied++
		}
	}

	// --- Reporte ---
	fmt.Printf("Productos scraper* evaluados: %d\n", total)
	fmt.Printf("Candidatos a cambio: %d  (colisiones skipeadas: %d)\n", len(changes)-len(collisions), len(collisions))
	fmt.Printf("Skips → no-resuelve: %d · ya-específico: %d · ya-correcto: %d\n\n", skipNoResuelve, skipYaEspecifico, skipMismo)

	fmt.Println("Cambios por rubro destino (relleno / corrección):")
	targets := make([]string, 0, len(byTarget))
	for t := range byTarget {
		targets = append(targets, t)
	}
	sort.Slice(targets, func(i, j int) bool {
		ti, tj := byTarget[targets[i]], byTarget[targets[j]]
		return (ti[0] + ti[1]) > (tj[0] + tj[1])
	})
	for _, t := range targets {
		agg := byTarget[t]
		fmt.Printf("  %-14s  total=%-5d  (relleno=%d, correccion=%d)\n", t, agg[0]+agg[1], agg[0], agg[1])
	}

	if len(collisions) > 0 {
		fmt.Printf("\nColisiones UNIQUE(name,business_type) — NO tocadas (%d):\n", len(collisions))
		shown := len(collisions)
		if shown > 15 {
			shown = 15
		}
		for _, c := range collisions[:shown] {
			fmt.Printf("  [%s→%s] %q (id=%s)\n", c.from, c.to, c.name, c.id)
		}
		if len(collisions) > shown {
			fmt.Printf("  ... y %d más\n", len(collisions)-shown)
		}
	}

	if *apply {
		fmt.Printf("\nAPLICADO: %d UPDATEs.\n", applied)
	} else {
		fmt.Printf("\nDRY-RUN: no se escribió nada. Correr con --apply para aplicar.\n")
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
