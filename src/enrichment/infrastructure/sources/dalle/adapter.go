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
	dalleAPIURL    = "https://api.openai.com/v1/images/generations"
	dalleModel     = "dall-e-3"
	dalleSize      = "1024x1024"
	dalleFormat    = "url"
	dalleHTTPTimeout = 60 * time.Second
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

func (s *DALLEGeneratorSource) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	if s.apiKey == "" {
		return nil, nil
	}

	imageURL, err := s.generateImage(ctx, name)
	if err != nil {
		return nil, err
	}

	if imageURL == "" {
		return nil, nil
	}

	return &entity.ProductData{
		ImageURL: imageURL,
		Source:   entity.SourceDALLE,
	}, nil
}

func (s *DALLEGeneratorSource) generateImage(ctx context.Context, name string) (string, error) {
	prompt := fmt.Sprintf(
		"professional product photo of %s, white background, studio lighting, no text, no labels",
		name,
	)

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
	payload := map[string]interface{}{
		"model":           dalleModel,
		"prompt":          prompt,
		"n":               1,
		"size":            dalleSize,
		"response_format": dalleFormat,
	}
	return json.Marshal(payload)
}

type dalleResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

func parseDALLEResponse(resp *http.Response) (string, error) {
	var result dalleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("dalle decoding response: %w", err)
	}

	if len(result.Data) == 0 || result.Data[0].URL == "" {
		return "", nil
	}

	return result.Data[0].URL, nil
}
