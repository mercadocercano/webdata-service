package perplexity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

const (
	perplexityAPIURL = "https://api.perplexity.ai/chat/completions"
	perplexityModel  = "sonar"
	httpTimeout      = 30 * time.Second
)

var gtinRegex = regexp.MustCompile(`\b\d{13}\b`)

type PerplexityGTINSource struct {
	apiKey     string
	httpClient *http.Client
	apiURL     string
	mlSource   port.ProductSource
}

func NewPerplexityGTINSource(mlSource port.ProductSource) *PerplexityGTINSource {
	return NewPerplexityGTINSourceWithURL(perplexityAPIURL, os.Getenv("PERPLEXITY_API_KEY"), mlSource)
}

// NewPerplexityGTINSourceWithURL permite inyectar URL y key para testing.
func NewPerplexityGTINSourceWithURL(apiURL, apiKey string, mlSource port.ProductSource) *PerplexityGTINSource {
	return &PerplexityGTINSource{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: httpTimeout},
		apiURL:     apiURL,
		mlSource:   mlSource,
	}
}

func (s *PerplexityGTINSource) SourceName() entity.Source {
	return entity.SourcePerplexity
}

func (s *PerplexityGTINSource) FindByGTIN(_ context.Context, _ string) (*entity.ProductData, error) {
	return nil, nil
}

func (s *PerplexityGTINSource) FindByNameWithContext(ctx context.Context, name, category, businessType string) (*entity.ProductData, error) {
	query := buildContextualQuery(name, category, businessType)
	return s.FindByName(ctx, query)
}

func buildContextualQuery(name, category, businessType string) string {
	query := name
	if category != "" {
		query += " " + category
	}
	if businessType != "" {
		query += " " + businessType
	}
	return strings.TrimSpace(query)
}

func (s *PerplexityGTINSource) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	if s.apiKey == "" {
		return nil, nil
	}

	gtin, err := s.searchGTIN(ctx, name)
	if err != nil {
		return nil, err
	}

	if gtin == "" {
		return nil, nil
	}

	return s.mlSource.FindByGTIN(ctx, gtin)
}

func (s *PerplexityGTINSource) searchGTIN(ctx context.Context, name string) (string, error) {
	prompt := fmt.Sprintf(
		"Find the EAN or GTIN-13 barcode number for this product: %s. Respond with only the 13-digit number, nothing else.",
		name,
	)

	body, err := buildRequestBody(prompt)
	if err != nil {
		return "", fmt.Errorf("perplexity building request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("perplexity creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perplexity request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	content, err := parsePerplexityResponse(resp)
	if err != nil {
		return "", err
	}

	return extractValidGTIN(content), nil
}

func buildRequestBody(prompt string) ([]byte, error) {
	payload := map[string]interface{}{
		"model": perplexityModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	return json.Marshal(payload)
}

type perplexityResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func parsePerplexityResponse(resp *http.Response) (string, error) {
	var result perplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("perplexity decoding response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", nil
	}

	return result.Choices[0].Message.Content, nil
}

func extractValidGTIN(text string) string {
	matches := gtinRegex.FindAllString(text, -1)
	for _, candidate := range matches {
		if isValidGTIN13(candidate) {
			return candidate
		}
	}
	return ""
}

// isValidGTIN13 valida check digit con algoritmo mod10 (GS1).
func isValidGTIN13(gtin string) bool {
	if len(gtin) != 13 {
		return false
	}

	sum := 0
	for i, ch := range gtin[:12] {
		digit := int(ch - '0')
		if i%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}

	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit == int(gtin[12]-'0')
}
