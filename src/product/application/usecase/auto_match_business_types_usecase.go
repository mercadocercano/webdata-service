package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

// AutoMatchRequest controls which products to match.
type AutoMatchRequest struct {
	ProductIDs             []uuid.UUID
	IncludeAlreadyAssigned bool
}

// BusinessTypeMatch represents a single proposed business type for a product.
type BusinessTypeMatch struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	MatchReason string `json:"match_reason"`
}

// AutoMatchProposal is a match proposal for one product.
type AutoMatchProposal struct {
	ProductID             uuid.UUID           `json:"product_id"`
	ProductTitle          string              `json:"product_title"`
	ProductImageURL       string              `json:"product_image_url,omitempty"`
	Category              string              `json:"category"`
	NormalizedCategory    string              `json:"normalized_category"`
	ProposedBusinessTypes []BusinessTypeMatch `json:"proposed_business_types"`
	ConfidenceScore       float64             `json:"confidence_score"`
}

// AutoMatchResult is the response from the auto-match use case.
type AutoMatchResult struct {
	Proposals      []AutoMatchProposal `json:"proposals"`
	TotalProducts  int                 `json:"total_products"`
	MatchedCount   int                 `json:"matched_count"`
	UnmatchedCount int                 `json:"unmatched_count"`
}

// categoryMapping maps normalized categories to business type keywords for matching.
// These keywords are matched against bt.Code and bt.Name to produce proposals.
// Rules:
//   - Only valid business type codes from the canonical map are used as keywords
//     (almacen, fiambreria, limpieza, kiosco, farmacia, perfumeria, supermercado,
//     bazar, ferreteria, electrodomesticos, ropa).
//   - No invalid codes: supermercado/autoservicio/vinoteca/carniceria/granja/
//     fruteria/pescaderia/dietetica/veterinaria/pet shop are removed.
//   - Taxonomía del owner: lácteos/leches/yogures/quesos → fiambreria (primer candidato).
var categoryMapping = map[string][]string{
	// Almacén de Barrio — alimentos secos, bebidas, consumo masivo
	"aceites":               {"almacen"},
	"arroz":                 {"almacen"},
	"fideos":                {"almacen"},
	"harinas":               {"almacen"},
	"azucar":                {"almacen"},
	"sal":                   {"almacen"},
	"conservas":             {"almacen"},
	"salsas":                {"almacen"},
	"condimentos":           {"almacen"},
	"especias":              {"almacen"},
	"snacks":                {"almacen", "kiosco"},
	"galletitas":            {"almacen", "kiosco"},
	"golosinas":             {"almacen", "kiosco"},
	"chocolates":            {"almacen", "kiosco"},
	"bebidas":               {"almacen"},
	"gaseosas":              {"almacen", "kiosco"},
	"aguas":                 {"almacen", "kiosco"},
	"jugos":                 {"almacen"},
	"cervezas":              {"almacen"},
	"vinos":                 {"almacen"},
	"congelados":            {"almacen"},
	"panaderia":             {"almacen"},
	"panificados":           {"almacen"},
	"cereales":              {"almacen"},
	"legumbres":             {"almacen"},
	"pastas":                {"almacen"},
	"cafe":                  {"almacen"},
	"te":                    {"almacen"},
	"yerba":                 {"almacen"},
	"infusiones":            {"almacen"},
	"dulces":                {"almacen"},
	"mermeladas":            {"almacen"},
	"almacen":               {"almacen"},

	// Fiambrería — taxonomía del owner: lácteos y sus derivados → fiambreria
	"lacteos":               {"fiambreria", "almacen"},
	"leches":                {"fiambreria", "almacen"},
	"yogures":               {"fiambreria"},
	"quesos":                {"fiambreria"},
	"fiambres":              {"fiambreria"},

	// Carnes, frutas, verduras — no tienen business type propio en el catálogo válido;
	// se propone almacen como mejor alternativa disponible.
	"carnes":                {"almacen"},
	"pollo":                 {"almacen"},
	"pescados":              {"almacen"},
	"frutas":                {"almacen"},
	"verduras":              {"almacen"},
	"mascotas":              {"almacen"},

	// Limpieza
	"limpieza":              {"limpieza"},
	"productos de limpieza": {"limpieza"},

	// Higiene / Perfumería / Cuidado personal
	"higiene":               {"perfumeria", "farmacia"},
	"perfumeria":            {"perfumeria", "farmacia"},
	"cuidado personal":      {"perfumeria", "farmacia"},
	"cosmeticos":            {"perfumeria", "farmacia"},

	// Farmacia
	"bebes":                 {"farmacia"},

	// Electrodomésticos
	"electronica":           {"electrodomesticos"},
	"tecnologia":            {"electrodomesticos"},
	"electro":               {"electrodomesticos"},
}

type AutoMatchBusinessTypesUseCase struct {
	repo     port.ProductRepository
	btProvider port.BusinessTypeProvider
}

func NewAutoMatchBusinessTypesUseCase(
	repo port.ProductRepository,
	btProvider port.BusinessTypeProvider,
) *AutoMatchBusinessTypesUseCase {
	return &AutoMatchBusinessTypesUseCase{repo: repo, btProvider: btProvider}
}

func (uc *AutoMatchBusinessTypesUseCase) Execute(
	ctx context.Context,
	tenantID uuid.UUID,
	req AutoMatchRequest,
) (*AutoMatchResult, error) {
	// 1. Fetch available business types from pim-service
	businessTypes, err := uc.btProvider.FetchAll(ctx, tenantID.String())
	if err != nil {
		return nil, err
	}

	// 2. Fetch products
	var products []*port.ProductSummary
	if len(req.ProductIDs) > 0 {
		products, err = uc.fetchByIDs(ctx, tenantID, req.ProductIDs, req.IncludeAlreadyAssigned)
	} else {
		products, err = uc.fetchAll(ctx, tenantID, req.IncludeAlreadyAssigned)
	}
	if err != nil {
		return nil, err
	}

	// 3. Match each product
	result := &AutoMatchResult{
		TotalProducts: len(products),
	}

	for _, p := range products {
		proposal := uc.matchProduct(p, businessTypes)
		if len(proposal.ProposedBusinessTypes) > 0 {
			result.Proposals = append(result.Proposals, proposal)
			result.MatchedCount++
		} else {
			result.UnmatchedCount++
		}
	}

	return result, nil
}

func (uc *AutoMatchBusinessTypesUseCase) fetchByIDs(
	ctx context.Context,
	tenantID uuid.UUID,
	ids []uuid.UUID,
	includeAssigned bool,
) ([]*port.ProductSummary, error) {
	var result []*port.ProductSummary
	for _, id := range ids {
		p, err := uc.repo.FindByID(ctx, tenantID, id)
		if err != nil {
			continue // skip not found
		}
		if !includeAssigned && len(p.BusinessTypes) > 0 {
			continue
		}
		result = append(result, &port.ProductSummary{
			ID:                 p.ID,
			Title:              p.Title,
			Category:           p.Category,
			NormalizedCategory: p.NormalizedCategory,
			Brand:              p.Brand,
			ImageURL:           p.ImageURL,
			BusinessTypes:      p.BusinessTypes,
		})
	}
	return result, nil
}

func (uc *AutoMatchBusinessTypesUseCase) fetchAll(
	ctx context.Context,
	tenantID uuid.UUID,
	includeAssigned bool,
) ([]*port.ProductSummary, error) {
	products, _, err := uc.repo.FindAll(ctx, tenantID, port.ProductFilter{
		Page:                 1,
		PageSize:             10000,
		WithoutBusinessTypes: !includeAssigned,
	})
	if err != nil {
		return nil, err
	}

	var result []*port.ProductSummary
	for _, p := range products {
		result = append(result, &port.ProductSummary{
			ID:                 p.ID,
			Title:              p.Title,
			Category:           p.Category,
			NormalizedCategory: p.NormalizedCategory,
			Brand:              p.Brand,
			ImageURL:           p.ImageURL,
			BusinessTypes:      p.BusinessTypes,
		})
	}
	return result, nil
}

func (uc *AutoMatchBusinessTypesUseCase) matchProduct(
	p *port.ProductSummary,
	businessTypes []port.BusinessType,
) AutoMatchProposal {
	proposal := AutoMatchProposal{
		ProductID:          p.ID,
		ProductTitle:       p.Title,
		ProductImageURL:    p.ImageURL,
		Category:           p.Category,
		NormalizedCategory: p.NormalizedCategory,
	}

	cat := stripAccents(strings.ToLower(strings.TrimSpace(p.NormalizedCategory)))
	if cat == "" {
		cat = stripAccents(strings.ToLower(strings.TrimSpace(p.Category)))
	}

	var keywords []string
	var matchSource string

	if cat != "" {
		keywords = findCategoryKeywords(cat)
		matchSource = "category_match: " + cat
	}

	// Fallback: match by title/brand keywords when no category
	if len(keywords) == 0 {
		keywords, matchSource = inferKeywordsFromTitleBrand(p.Title, p.Brand)
	}

	if len(keywords) == 0 {
		return proposal
	}

	// Match keywords against available business types
	var matches []BusinessTypeMatch
	for _, kw := range keywords {
		for _, bt := range businessTypes {
			btNameLower := strings.ToLower(bt.Name)
			btCodeLower := strings.ToLower(bt.Code)
			if strings.Contains(btNameLower, kw) || strings.Contains(btCodeLower, kw) {
				matches = append(matches, BusinessTypeMatch{
					Code:        bt.Code,
					Name:        bt.Name,
					MatchReason: matchSource + " → " + kw,
				})
				break // one match per keyword
			}
		}
		if len(matches) >= 3 {
			break // max 3 options
		}
	}

	if len(matches) > 0 {
		proposal.ProposedBusinessTypes = matches
		if cat != "" && len(matches) >= 2 {
			proposal.ConfidenceScore = 0.9
		} else if cat != "" {
			proposal.ConfidenceScore = 0.7
		} else {
			proposal.ConfidenceScore = 0.5 // lower confidence for title/brand inference
		}
	}

	return proposal
}

// titleBrandSignals maps keywords found in product titles/brands to business type keywords.
var titleBrandSignals = map[string][]string{
	// Farmacia / Perfumería signals
	"serum":       {"farmacia", "perfumeria"},
	"crema":       {"farmacia", "perfumeria"},
	"protector solar": {"farmacia", "perfumeria"},
	"shampoo":     {"farmacia", "perfumeria", "supermercado"},
	"acondicionador": {"farmacia", "perfumeria"},
	"micelar":     {"farmacia", "perfumeria"},
	"hidratacion": {"farmacia", "perfumeria"},
	"facial":      {"farmacia", "perfumeria"},
	"capilar":     {"farmacia", "perfumeria"},
	"corporal":    {"farmacia", "perfumeria"},
	"desodorante": {"farmacia", "perfumeria", "supermercado"},
	"labial":      {"farmacia", "perfumeria"},
	"maquillaje":  {"farmacia", "perfumeria"},
	"vitamina":    {"farmacia"},
	"fps":         {"farmacia", "perfumeria"},
	// Brand signals
	"garnier":      {"farmacia", "perfumeria"},
	"loreal":       {"farmacia", "perfumeria"},
	"vichy":        {"farmacia", "perfumeria"},
	"la roche":     {"farmacia", "perfumeria"},
	"cerave":       {"farmacia", "perfumeria"},
	"maybelline":   {"farmacia", "perfumeria"},
	"neutrogena":   {"farmacia", "perfumeria"},
	"caviahue":     {"farmacia", "perfumeria"},
	"eucerin":      {"farmacia", "perfumeria"},
	"nivea":        {"farmacia", "perfumeria", "supermercado"},
	// Electronica signals
	"heladera":     {"electronica"},
	"lavarropas":   {"electronica"},
	"microondas":   {"electronica"},
	"aire acondicionado": {"electronica"},
	"freezer":      {"electronica"},
	"freidora":     {"electronica"},
	"aspiradora":   {"electronica"},
	"notebook":     {"electronica"},
	"celular":      {"electronica"},
	"smart tv":     {"electronica"},
	"led":          {"electronica"},
	// Hair care
	"balsamo":      {"farmacia", "perfumeria", "supermercado"},
	"capilatis":    {"farmacia", "perfumeria"},
	// Personal care
	"preservativo": {"farmacia"},
	"preservativos": {"farmacia"},
}

// stripAccents replaces common Spanish/Portuguese accented characters with ASCII equivalents.
func stripAccents(s string) string {
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u",
		"Ü", "u", "Ñ", "n",
	)
	return r.Replace(s)
}

// inferKeywordsFromTitleBrand tries to determine business type keywords from
// product title and brand when category is not available.
func inferKeywordsFromTitleBrand(title, brand string) ([]string, string) {
	combined := stripAccents(strings.ToLower(title + " " + brand))

	for signal, keywords := range titleBrandSignals {
		if strings.Contains(combined, signal) {
			return keywords, "title_brand_match: " + signal
		}
	}

	return nil, ""
}

// findCategoryKeywords looks up a category in the mapping table.
// Falls back to fuzzy (contains) matching if exact match fails.
func findCategoryKeywords(cat string) []string {
	// Exact match
	if keywords, ok := categoryMapping[cat]; ok {
		return keywords
	}

	// Contains match — require minimum 4-char keys to avoid false positives
	// (e.g., "te" matching inside "tecnología")
	for key, keywords := range categoryMapping {
		if len(key) < 4 {
			continue
		}
		if strings.Contains(cat, key) || strings.Contains(key, cat) {
			return keywords
		}
	}

	return nil
}
