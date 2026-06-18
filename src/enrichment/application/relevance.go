package application

import "strings"

// MinRelevanceScore es la cobertura mínima de tokens significativos del NOMBRE
// del producto que deben aparecer en el TÍTULO devuelto por una fuente real
// (MercadoLibre, OpenFoodFacts, Firecrawl, Perplexity) para aceptar su imagen.
//
// Motivación (E16, piloto 2026-06-18): ML hace keyword-match de nombres
// ambiguos y devuelve imágenes con confianza pero equivocadas — "Aguja x kg"
// (corte vacuno) → caja de jeringas; "Alitas de pollo" → salsa para alitas.
// Como cuesta 0¢ y "tiene éxito", se escribía basura en silencio. Este guard
// rechaza esos matches y deja que la cascada caiga a DALL-E (último recurso
// sintético, exento del guard).
//
// Umbral 0.6: conserva los matches de productos de marca (B1, p.ej. "Auto Hot
// Wheels básico surtido" ≈ título ML "Hot Wheels Auto Básico Surtido"), y
// rechaza los genéricos/ambiguos (B2) → van a DALL-E.
const MinRelevanceScore = 0.6

// stopwords: conectores, unidades y cuantificadores que no aportan a la
// identidad del producto y no deben contar para el match.
var stopwords = map[string]bool{
	"de": true, "del": true, "la": true, "el": true, "los": true, "las": true,
	"con": true, "sin": true, "por": true, "para": true, "y": true, "a": true,
	"x": true, "un": true, "una": true, "u": true, "uds": true, "unidad": true,
	"unidades": true, "docena": true, "pack": true,
	"kg": true, "g": true, "gr": true, "grs": true, "mg": true,
	"l": true, "lt": true, "ml": true, "cc": true, "cm": true, "mm": true,
}

// IsRelevantMatch indica si el título de la fuente matchea lo suficiente con el
// nombre del producto. Si el nombre no tiene tokens significativos (no se puede
// validar), retorna true para no sobre-rechazar.
func IsRelevantMatch(productName, sourceTitle string) bool {
	return RelevanceScore(productName, sourceTitle) >= MinRelevanceScore
}

// RelevanceScore = |tokens_significativos(nombre) ∩ tokens(título)| / |tokens_significativos(nombre)|.
// Retorna 1.0 cuando el nombre no tiene tokens significativos (no validable).
func RelevanceScore(productName, sourceTitle string) float64 {
	query := significantTokens(productName)
	if len(query) == 0 {
		return 1.0
	}
	title := tokenSet(sourceTitle)

	matched := 0
	for tok := range query {
		if title[tok] {
			matched++
		}
	}
	return float64(matched) / float64(len(query))
}

// significantTokens devuelve el set de tokens del texto sin stopwords ni números puros.
func significantTokens(s string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range tokenize(s) {
		if stopwords[tok] || isNumeric(tok) {
			continue
		}
		out[tok] = true
	}
	return out
}

// tokenSet devuelve TODOS los tokens del texto (sin filtrar), para buscar coincidencias.
func tokenSet(s string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range tokenize(s) {
		out[tok] = true
	}
	return out
}

// tokenize normaliza (minúsculas, sin acentos) y separa por cualquier caracter no alfanumérico.
func tokenize(s string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		b.WriteRune(foldAccent(r))
	}
	return strings.FieldsFunc(b.String(), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
}

func isNumeric(tok string) bool {
	for _, r := range tok {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(tok) > 0
}

// foldAccent mapea acentos del español a su letra base (evita dependencia de x/text).
func foldAccent(r rune) rune {
	switch r {
	case 'á', 'à', 'ä', 'â':
		return 'a'
	case 'é', 'è', 'ë', 'ê':
		return 'e'
	case 'í', 'ì', 'ï', 'î':
		return 'i'
	case 'ó', 'ò', 'ö', 'ô':
		return 'o'
	case 'ú', 'ù', 'ü', 'û':
		return 'u'
	case 'ñ':
		return 'n'
	}
	return r
}
