package value_object

import (
	"strings"
)

// productCategoryKeyword maps a normalized keyword (lowercase, no accents) to a
// business type code and name. Must stay aligned with categoryToBusinessType in
// category_business_type_map.go — same valid codes/names, same taxonomy decisions
// (e.g. lacteos → fiambreria, limpieza → limpieza, supermercado/almacen → almacen).
//
// Keys are single words; matching is done via strings.Contains so a keyword "leche"
// will match both "leches larga vida" and "/Lácteos/Leches/".
// Order matters only within a keyword group when building the result; the lookup
// iterates keywords in declaration order via the slice below.
type productCategoryRule struct {
	keyword          string
	businessTypeCode string
	businessTypeName string
}

// productCategoryRules is evaluated in order. The FIRST rule whose keyword is found
// inside the normalized category wins. Put more specific keywords before broader ones
// when two could match the same string (e.g. "yogur" before "lacteo").
var productCategoryRules = []productCategoryRule{
	// --- Fiambrería (taxonomía: lácteos + fiambres/quesos van a fiambrería) ---
	{keyword: "yogur", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "queso", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "fiambre", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "lacteo", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "lacteos", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "leche", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "manteca", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "crema", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},
	{keyword: "margarina", businessTypeCode: "fiambreria", businessTypeName: "Fiambrería"},

	// --- Limpieza ---
	{keyword: "limpieza", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},
	{keyword: "lavandina", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},
	{keyword: "detergente", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},
	{keyword: "cuidado de ropa", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},
	{keyword: "ropa lavado", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},
	{keyword: "suavizante", businessTypeCode: "limpieza", businessTypeName: "Limpieza"},

	// --- Farmacia ---
	{keyword: "farmacia", businessTypeCode: "farmacia", businessTypeName: "Farmacia"},
	{keyword: "bebe", businessTypeCode: "farmacia", businessTypeName: "Farmacia"},
	{keyword: "bebes", businessTypeCode: "farmacia", businessTypeName: "Farmacia"},
	{keyword: "pañal", businessTypeCode: "farmacia", businessTypeName: "Farmacia"},
	{keyword: "panal", businessTypeCode: "farmacia", businessTypeName: "Farmacia"},

	// --- Perfumería ---
	{keyword: "perfumeria", businessTypeCode: "perfumeria", businessTypeName: "Perfumería"},
	{keyword: "higiene personal", businessTypeCode: "perfumeria", businessTypeName: "Perfumería"},
	{keyword: "cuidado personal", businessTypeCode: "perfumeria", businessTypeName: "Perfumería"},
	{keyword: "shampoo", businessTypeCode: "perfumeria", businessTypeName: "Perfumería"},
	{keyword: "desodorante", businessTypeCode: "perfumeria", businessTypeName: "Perfumería"},

	// --- Ferretería ---
	{keyword: "ferreteria", businessTypeCode: "ferreteria", businessTypeName: "Ferretería"},
	{keyword: "construccion", businessTypeCode: "ferreteria", businessTypeName: "Ferretería"},
	{keyword: "sanitario", businessTypeCode: "ferreteria", businessTypeName: "Ferretería"},

	// --- Casa de Electrodomésticos ---
	{keyword: "electrodomestico", businessTypeCode: "electrodomesticos", businessTypeName: "Casa de Electrodomésticos"},
	{keyword: "electronica", businessTypeCode: "electrodomesticos", businessTypeName: "Casa de Electrodomésticos"},
	{keyword: "tecnologia", businessTypeCode: "electrodomesticos", businessTypeName: "Casa de Electrodomésticos"},

	// --- Bazar ---
	{keyword: "bazar", businessTypeCode: "bazar", businessTypeName: "Bazar"},

	// --- Tienda de Ropa ---
	{keyword: "indumentaria", businessTypeCode: "ropa", businessTypeName: "Tienda de Ropa"},
	{keyword: "calzado", businessTypeCode: "ropa", businessTypeName: "Tienda de Ropa"},

	// --- Almacén de Barrio (catch-all de alimentos/bebidas/consumo masivo) ---
	// Bebidas
	{keyword: "bebida", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "gaseosa", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "agua", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "jugo", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "cerveza", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "vino", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	// Desayuno / infusiones
	{keyword: "cafe", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "cafe", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "infusion", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "yerba", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "mate", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "te ", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"}, // trailing space to avoid "tecnologia"
	// Aceites / condimentos
	{keyword: "aceite", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "vinagre", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "condimento", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "salsa", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "conserva", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "mermelada", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	// Harinas / granos
	{keyword: "harina", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "arroz", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "fideo", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "pasta", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "cereal", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "legumbre", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "azucar", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "sal", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	// Panadería / galletitas
	{keyword: "galletita", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "pan ", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"}, // trailing space
	{keyword: "panaderia", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	// Golosinas / chocolates / snacks (kiosco también aplica; ante duda → almacen per spec)
	{keyword: "chocolate", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "golosina", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "alfajor", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "snack", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	// Supermercado / almacen genérico
	{keyword: "almacen", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "supermercado", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
	{keyword: "autoservicio", businessTypeCode: "almacen", businessTypeName: "Almacén de Barrio"},
}

// normalizeProductCategory returns a lowercase, accent-stripped, trimmed version
// of the raw category string. Path-style categories like "/Lácteos/Yogures/Yogur en vasos/"
// have their slashes replaced with spaces so keyword matching works across all segments.
func normalizeProductCategory(raw string) string {
	// Replace path separators with spaces so all segments are searchable.
	s := strings.ReplaceAll(raw, "/", " ")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = stripProductAccents(s)
	return s
}

// stripProductAccents replaces common Spanish accented characters with ASCII.
// Deliberately kept local to this file to avoid coupling with the usecase package's
// stripAccents function.
func stripProductAccents(s string) string {
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u",
		"Ü", "u", "Ñ", "n",
	)
	return r.Replace(s)
}

// ResolveBusinessTypeFromProductCategory resolves the business type for an individual
// product's raw category string (e.g. "Cafés", "/Lácteos/Yogures/Yogur en vasos/",
// "LIMPIEZA"). It normalizes the input and matches against productCategoryRules.
//
// Returns (assignment, true) when a match is found, (zero, false) otherwise.
// Callers should fall back to the source-level autoAssignment when this returns false.
func ResolveBusinessTypeFromProductCategory(rawCategory string) (BusinessTypeAssignment, bool) {
	if rawCategory == "" {
		return BusinessTypeAssignment{}, false
	}

	normalized := normalizeProductCategory(rawCategory)
	if normalized == "" {
		return BusinessTypeAssignment{}, false
	}

	// Pad with a leading and trailing space so boundary-sensitive keywords
	// like "te " and "pan " can match at start-of-string or segment boundaries
	// without false positives inside longer words.
	padded := " " + normalized + " "

	for _, rule := range productCategoryRules {
		if strings.Contains(padded, rule.keyword) {
			assignment, err := NewBusinessTypeAssignment(rule.businessTypeCode, rule.businessTypeName)
			if err != nil {
				return BusinessTypeAssignment{}, false
			}
			return assignment, true
		}
	}

	return BusinessTypeAssignment{}, false
}
