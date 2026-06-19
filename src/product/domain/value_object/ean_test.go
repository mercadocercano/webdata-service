package value_object

import "testing"

func TestIsValidEAN13(t *testing.T) {
	cases := []struct {
		in   string
		want bool
		desc string
	}{
		{"7790070410917", true, "13 dígitos + checksum correcto"},
		{" 7790070410917 ", true, "EAN válido con espacios — trim ok"},
		{"7790742044921", true, "La Serenísima — EAN-13 válido real"},
		{"230415", false, "PLU corto de pesable (6 dígitos)"},
		{"210165", false, "código interno VTEX corto — caso real E19"},
		{"", false, "vacío"},
		{"779007041091", false, "12 dígitos — len falla"},
		{"77900704109170", false, "14 dígitos — len falla"},
		{"779007041091X", false, "no numérico"},
		{"7891167022278", false, "13 dígitos, checksum inválido (dato corrupto retailer)"},
		{"7796074860226", false, "13 dígitos, checksum inválido — caso exacto del log E19 (esperado=2, got=6)"},
		{"7790130003465", false, "13 dígitos, checksum inválido"},
		{"0779915000814", false, "13 dígitos con cero inicial, checksum inválido"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := IsValidEAN13(c.in); got != c.want {
				t.Errorf("IsValidEAN13(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestNormalizeEANForSync(t *testing.T) {
	// Tabla exhaustiva — cubre exactamente los casos observados en E19.
	// Invariante: un EAN inválido NUNCA debe llegar al PIM; se reemplaza por "".
	// Esto permite que el producto sincronice y caiga en dedup por nombre+marca.
	cases := []struct {
		in   string
		want string
		desc string
	}{
		// --- Casos inválidos que deben resultar en "" ---
		{"210165", "", "código interno VTEX corto — caso real E19"},
		{"7796074860226", "", "13 dígitos checksum inválido — caso exacto del log E19"},
		{"", "", "EAN vacío — pasa como vacío sin error"},
		{"7891167022278", "", "13 dígitos checksum inválido — dato corrupto retailer"},
		// --- Casos válidos que deben pasar intactos ---
		{"7790742044921", "7790742044921", "La Serenísima — EAN-13 válido pasa sin modificar"},
		{"7790070410917", "7790070410917", "EAN-13 válido genérico"},
		{" 7790070410917 ", "7790070410917", "EAN-13 válido con espacios — trim aplicado"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := NormalizeEANForSync(c.in)
			if got != c.want {
				t.Errorf("NormalizeEANForSync(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
