package firecrawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

const agentEndpoint = "https://api.firecrawl.dev/v1/agent"

type FirecrawlEnrichAdapter struct {
	apiKey     string
	httpClient *http.Client
}

func NewFirecrawlEnrichAdapter() *FirecrawlEnrichAdapter {
	return &FirecrawlEnrichAdapter{
		apiKey:     os.Getenv("FIRECRAWL_API_KEY"),
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (a *FirecrawlEnrichAdapter) SourceName() entity.Source {
	return entity.SourceFirecrawl
}

func (a *FirecrawlEnrichAdapter) FindByGTIN(ctx context.Context, gtin string) (*entity.ProductData, error) {
	if a.apiKey == "" {
		return nil, nil
	}
	return a.agentSearch(ctx, gtin)
}

func (a *FirecrawlEnrichAdapter) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	if a.apiKey == "" {
		return nil, nil
	}
	return a.agentSearch(ctx, name)
}

func (a *FirecrawlEnrichAdapter) agentSearch(ctx context.Context, query string) (*entity.ProductData, error) {
	prompt := fmt.Sprintf(
		"Find official product image URL, brand and category for: %s. Return JSON only with fields image_url, brand, category. No markdown.",
		query,
	)

	payload, err := json.Marshal(map[string]string{
		"model":  "fire-1",
		"prompt": prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("firecrawl marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agentEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("firecrawl creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firecrawl request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseFirecrawlAgentResponse(resp.Body, query)
}

type agentResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
}

type agentProductData struct {
	ImageURL string `json:"image_url"`
	Brand    string `json:"brand"`
	Category string `json:"category"`
}

func parseFirecrawlAgentResponse(body io.Reader, query string) (*entity.ProductData, error) {
	var response agentResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding firecrawl response: %w", err)
	}

	if !response.Success || response.Data == "" {
		return nil, nil
	}

	jsonStr := extractJSON(response.Data)
	if jsonStr == "" {
		return nil, nil
	}

	var productData agentProductData
	if err := json.Unmarshal([]byte(jsonStr), &productData); err != nil {
		return nil, fmt.Errorf("parsing firecrawl agent data: %w", err)
	}

	if productData.ImageURL == "" {
		return nil, nil
	}

	return &entity.ProductData{
		Name:     query,
		Brand:    productData.Brand,
		Category: productData.Category,
		ImageURL: productData.ImageURL,
		Source:   entity.SourceFirecrawl,
	}, nil
}

// extractJSON extrae JSON de una string que puede venir con markdown fences.
func extractJSON(data string) string {
	data = strings.TrimSpace(data)

	// Quitar markdown fence si existe
	if strings.HasPrefix(data, "```") {
		lines := strings.Split(data, "\n")
		if len(lines) < 2 {
			return ""
		}
		// Primera linea es ``` o ```json, ultima es ```
		inner := lines[1 : len(lines)-1]
		data = strings.Join(inner, "\n")
	}

	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "{") {
		return data
	}
	return ""
}
