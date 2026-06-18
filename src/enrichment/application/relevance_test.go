package application

import (
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

func TestIsGenericBulk(t *testing.T) {
	cases := []struct {
		name  string
		item  port.NeedsEnrichmentItem
		want  bool
	}{
		{"sin marca", port.NeedsEnrichmentItem{Name: "Asado de tira", Brand: ""}, true},
		{"por peso con marca", port.NeedsEnrichmentItem{Name: "Alas de pollo Granja Tres Arroyos x kg", Brand: "Granja Tres Arroyos"}, true},
		{"aguja granel", port.NeedsEnrichmentItem{Name: "Aguja x kg", Brand: ""}, true},
		{"marca empaquetado (B1)", port.NeedsEnrichmentItem{Name: "Auto Hot Wheels básico surtido", Brand: "Hot Wheels"}, false},
		{"barbie B1", port.NeedsEnrichmentItem{Name: "Barbie Dreamhouse aventuras", Brand: "Mattel"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isGenericBulk(c.item); got != c.want {
				t.Errorf("isGenericBulk(%q, brand=%q) = %v, want %v", c.item.Name, c.item.Brand, got, c.want)
			}
		})
	}
}

func TestIsRelevantMatch(t *testing.T) {
	cases := []struct {
		name    string
		product string
		title   string
		want    bool
	}{
		// B1 (marca) — deben ACEPTARSE
		{"hotwheels", "Auto Hot Wheels básico surtido", "Hot Wheels Auto Básico Surtido Mattel", true},
		{"abaco", "Abaco de madera didáctico", "Abaco Didáctico de Madera Infantil", true},
		{"barbie", "Barbie Dreamhouse aventuras", "Muñeca Barbie Dreamhouse Adventures Mattel", true},
		{"corte exacto", "Lomo", "Lomo de cerdo fresco x kg", true},

		// B2 / mismatches reales del piloto — deben RECHAZARSE
		{"aguja->jeringa", "Aguja x kg", "NIPRO Syringe 10ml with needle x100", false},
		{"alitas->salsa", "Alitas de pollo", "First Street Chicken Wing Sauce salsa", false},

		// nombre sin tokens significativos → no validable → ACEPTA
		{"solo unidades", "x kg", "lo que sea", true},
		{"nombre vacio", "", "cualquier titulo", true},
		{"titulo vacio", "Asado de tira", "", false},
		{"nombre todo numeros", "12345", "producto 12345", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsRelevantMatch(c.product, c.title); got != c.want {
				t.Errorf("IsRelevantMatch(%q, %q) = %v (score %.2f), want %v",
					c.product, c.title, got, RelevanceScore(c.product, c.title), c.want)
			}
		})
	}
}
