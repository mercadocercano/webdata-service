package value_object

import "testing"

func TestIsValidEAN13(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"7790070410917", true},  // 13 dígitos + checksum correcto
		{" 7790070410917 ", true},
		{"230415", false},         // PLU corto de pesable
		{"", false},               //
		{"779007041091", false},   // 12 dígitos
		{"77900704109170", false}, // 14 dígitos
		{"779007041091X", false},  // no numérico
		{"7891167022278", false},  // 13 dígitos, checksum inválido (dato corrupto retailer)
		{"7790130003465", false},  // 13 dígitos, checksum inválido
		{"0779915000814", false},  // 13 dígitos con cero inicial, checksum inválido
	}
	for _, c := range cases {
		if got := IsValidEAN13(c.in); got != c.want {
			t.Errorf("IsValidEAN13(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNormalizeEANForSync(t *testing.T) {
	if got := NormalizeEANForSync("7790070410917"); got != "7790070410917" {
		t.Errorf("EAN válido debería pasar, got %q", got)
	}
	if got := NormalizeEANForSync(" 7790070410917 "); got != "7790070410917" {
		t.Errorf("EAN válido debería trimear, got %q", got)
	}
	if got := NormalizeEANForSync("230415"); got != "" {
		t.Errorf("PLU inválido debería caer a vacío, got %q", got)
	}
	if got := NormalizeEANForSync(""); got != "" {
		t.Errorf("vacío debería seguir vacío, got %q", got)
	}
}
