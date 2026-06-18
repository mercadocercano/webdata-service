package dalle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

const (
	dalleAPIURL = "https://api.openai.com/v1/images/generations"
	// gpt-image-1: modelo de imagen vigente de OpenAI (dall-e-2/3 no están
	// habilitados en el proyecto de la API key). Devuelve b64_json, no URL.
	dalleModel       = "gpt-image-1"
	dalleSize        = "1024x1024"
	dalleQuality     = "medium"
	dalleHTTPTimeout = 120 * time.Second
)

type DALLEGeneratorSource struct {
	apiKey     string
	httpClient *http.Client
	apiURL     string
}

func NewDALLEGeneratorSource() *DALLEGeneratorSource {
	return NewDALLEGeneratorSourceWithURL(dalleAPIURL, os.Getenv("OPENAI_API_KEY"))
}

// NewDALLEGeneratorSourceWithURL permite inyectar URL y key para testing.
func NewDALLEGeneratorSourceWithURL(apiURL, apiKey string) *DALLEGeneratorSource {
	return &DALLEGeneratorSource{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: dalleHTTPTimeout},
		apiURL:     apiURL,
	}
}

func (s *DALLEGeneratorSource) SourceName() entity.Source {
	return entity.SourceDALLE
}

func (s *DALLEGeneratorSource) FindByGTIN(_ context.Context, _ string) (*entity.ProductData, error) {
	return nil, nil
}

func (s *DALLEGeneratorSource) FindByNameWithContext(ctx context.Context, name, category, businessType string) (*entity.ProductData, error) {
	return s.generate(ctx, name, category, businessType)
}

func (s *DALLEGeneratorSource) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	return s.generate(ctx, name, "", "")
}

func (s *DALLEGeneratorSource) generate(ctx context.Context, name, category, businessType string) (*entity.ProductData, error) {
	if s.apiKey == "" {
		return nil, nil
	}

	imageURL, err := s.generateImage(ctx, buildPrompt(name, category, businessType))
	if err != nil {
		return nil, err
	}

	if imageURL == "" {
		return nil, nil
	}

	return &entity.ProductData{
		Name:     name,
		ImageURL: imageURL,
		Source:   entity.SourceDALLE,
	}, nil
}

// buildPrompt arma un prompt de FOTO REALISTA con contexto de rubro. El contexto
// desambigua nombres genéricos del catálogo argentino: sin él, "Aguja x kg"
// (corte vacuno) genera una aguja de coser/jeringa; con "carnicería" genera carne.
func buildPrompt(name, category, businessType string) string {
	subject := cleanProductName(name)

	var ctxParts []string
	if c := businessContext(businessType); c != "" {
		ctxParts = append(ctxParts, c)
	}
	if category != "" {
		ctxParts = append(ctxParts, "category: "+category)
	}
	ctxPhrase := ""
	if len(ctxParts) > 0 {
		ctxPhrase = " (" + strings.Join(ctxParts, "; ") + ")"
	}

	return fmt.Sprintf(
		"Realistic professional product photograph of \"%s\"%s, a product sold in retail stores in Argentina. "+
			"Centered on a clean white background, soft studio lighting, sharp focus, high detail, "+
			"photorealistic, appetizing, true to life. No text, no logos, no watermark, no packaging mockup.",
		subject, ctxPhrase,
	)
}

// businessContext mapea el business_type a una pista de dominio en inglés para DALL-E.
func businessContext(businessType string) string {
	switch businessType {
	case "carniceria":
		return "fresh raw butcher meat cut"
	case "verduleria":
		return "fresh fruit or vegetable from a greengrocer"
	case "fiambreria":
		return "deli cold cut or dairy product"
	case "panaderia":
		return "bakery product"
	case "pescaderia":
		return "fresh fish or seafood"
	case "vinoteca":
		return "wine or spirits bottle"
	case "jugueteria":
		return "children's toy"
	case "libreria":
		return "stationery or bookstore item"
	case "ferreteria":
		return "hardware tool or item"
	case "":
		return ""
	default:
		return "product typically sold at a " + businessType + " store"
	}
}

// cleanProductName quita cuantificadores de venta por peso/unidad ("x kg", "x1kg",
// "x4", "500g") que ensucian el prompt sin aportar a la identidad visual.
func cleanProductName(name string) string {
	tokens := strings.Fields(name)
	kept := make([]string, 0, len(tokens))
	for _, t := range tokens {
		lt := strings.ToLower(t)
		if lt == "x" {
			continue
		}
		if isSaleQuantityToken(lt) {
			continue
		}
		kept = append(kept, t)
	}
	out := strings.Join(kept, " ")
	if len(kept) == 0 {
		out = strings.TrimSpace(name)
	}
	// Cota de longitud: evita prompts gigantes / texto adversarial en el nombre.
	if len(out) > 100 {
		out = strings.TrimSpace(out[:100])
	}
	return out
}

// isSaleQuantityToken detecta tokens tipo "kg","500g","1kg","x4","473cc","2u".
func isSaleQuantityToken(tok string) bool {
	tok = strings.TrimPrefix(tok, "x")
	hasDigit, hasUnit := false, false
	rest := tok
	for _, u := range []string{"kg", "grs", "gr", "g", "mg", "lt", "l", "ml", "cc", "u", "un"} {
		if strings.HasSuffix(rest, u) {
			rest = strings.TrimSuffix(rest, u)
			hasUnit = true
			break
		}
	}
	if rest == "" {
		return hasUnit // "kg", "l", "u" puro
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return false
		}
		hasDigit = true
	}
	return hasDigit // "500g", "x4" (sin unidad), "1kg"
}

func (s *DALLEGeneratorSource) generateImage(ctx context.Context, prompt string) (string, error) {
	body, err := buildDALLERequest(prompt)
	if err != nil {
		return "", fmt.Errorf("dalle building request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("dalle creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("dalle request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("dalle API returned status %d", resp.StatusCode)
	}

	return parseDALLEResponse(resp)
}

func buildDALLERequest(prompt string) ([]byte, error) {
	// gpt-image-1 NO acepta response_format y siempre devuelve b64_json.
	payload := map[string]interface{}{
		"model":   dalleModel,
		"prompt":  prompt,
		"n":       1,
		"size":    dalleSize,
		"quality": dalleQuality,
	}
	return json.Marshal(payload)
}

type dalleResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

// parseDALLEResponse devuelve la imagen como data URI (gpt-image-1 entrega base64,
// no URL). El downstream (downloadURL) decodifica el data URI a bytes.
func parseDALLEResponse(resp *http.Response) (string, error) {
	var result dalleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("dalle decoding response: %w", err)
	}

	if len(result.Data) == 0 {
		return "", nil
	}
	if result.Data[0].B64JSON != "" {
		return "data:image/png;base64," + result.Data[0].B64JSON, nil
	}
	// Compat: si algún modelo devolviera URL.
	return result.Data[0].URL, nil
}
