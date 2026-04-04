package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

const (
	firecrawlBaseURL    = "https://api.firecrawl.dev/v1"
	maxRetries          = 3
	extractPollInterval = 5 * time.Second
)

type FirecrawlAdapter struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func parseDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

func NewFirecrawlAdapter(apiKey string) *FirecrawlAdapter {
	return &FirecrawlAdapter{
		apiKey:       apiKey,
		baseURL:      firecrawlBaseURL,
		pollInterval: extractPollInterval,
		pollTimeout:  parseDurationEnv("FIRECRAWL_EXTRACT_TIMEOUT", 5*time.Minute),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// newFirecrawlAdapterWithBaseURL creates an adapter with a custom base URL and poll interval.
// Intended for use in tests only.
func newFirecrawlAdapterWithBaseURL(apiKey, baseURL string) *FirecrawlAdapter {
	return &FirecrawlAdapter{
		apiKey:       apiKey,
		baseURL:      baseURL,
		pollInterval: 100 * time.Millisecond, // fast polling for tests
		pollTimeout:  10 * time.Second,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Extract calls the async /v1/extract endpoint, polls until completion,
// and returns the structured product list.
func (a *FirecrawlAdapter) Extract(ctx context.Context, url string, schema json.RawMessage, opts scrapingport.ExtractOptions) ([]scrapingport.RawProduct, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("firecrawl API key not configured")
	}

	payload := map[string]interface{}{
		"urls":   []string{url},
		"schema": schema,
	}
	if opts.Prompt != "" {
		payload["prompt"] = opts.Prompt
	}

	// Bug 1 fix: POST /extract returns an async job id, not the result directly.
	initBody, err := a.doRequestWithRetry(ctx, "POST", "/extract", payload)
	if err != nil {
		return nil, fmt.Errorf("firecrawl extract submit: %w", err)
	}

	var initResp struct {
		Success bool   `json:"success"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(initBody, &initResp); err != nil {
		return nil, fmt.Errorf("parsing extract init response: %w", err)
	}
	if !initResp.Success || initResp.ID == "" {
		return nil, fmt.Errorf("firecrawl extract did not return a job id")
	}

	return a.pollExtractJob(ctx, initResp.ID)
}

// pollExtractJob polls GET /v1/extract/{id} until status is "completed".
func (a *FirecrawlAdapter) pollExtractJob(ctx context.Context, jobID string) ([]scrapingport.RawProduct, error) {
	deadline := time.Now().Add(a.pollTimeout)
	path := "/extract/" + jobID

	// Bug 2 fix: data is a single object, not an array.
	// Bug 3 fix: product name field is "name" in the schema, not "title".
	type productItem struct {
		Title         string   `json:"name"`
		Price         *float64 `json:"price"`
		OriginalPrice *float64 `json:"original_price"`
		URL           string   `json:"url"`
		ImageURL      string   `json:"image_url"`
		Description   string   `json:"description"`
		Brand         string   `json:"brand"`
		Category      string   `json:"category"`
		SKU           string   `json:"sku"`
		InStock       *bool    `json:"in_stock"`
	}

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("firecrawl extract job %s timed out after %s", jobID, a.pollTimeout)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(a.pollInterval):
		}

		pollBody, err := a.doRequestWithRetry(ctx, "GET", path, nil)
		if err != nil {
			return nil, fmt.Errorf("polling extract job %s: %w", jobID, err)
		}

		// Bug 2 fix: during "processing" the API returns data:[] (empty array),
		// only when "completed" does it return data:{products:[...]} (object).
		// Use RawMessage to defer parsing until we confirm status=completed.
		var pollResp struct {
			Status string          `json:"status"`
			Data   json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(pollBody, &pollResp); err != nil {
			return nil, fmt.Errorf("parsing extract poll response: %w", err)
		}

		if pollResp.Status != "completed" {
			continue
		}

		var extractedData struct {
			Products []productItem `json:"products"`
		}
		if err := json.Unmarshal(pollResp.Data, &extractedData); err != nil {
			return nil, fmt.Errorf("parsing extract data: %w", err)
		}

		products := make([]scrapingport.RawProduct, 0, len(extractedData.Products))
		for _, p := range extractedData.Products {
			products = append(products, scrapingport.RawProduct{
				Title:         p.Title,
				Price:         p.Price,
				OriginalPrice: p.OriginalPrice,
				URL:           p.URL,
				ImageURL:      p.ImageURL,
				Description:   p.Description,
				Brand:         p.Brand,
				Category:      p.Category,
				SKU:           p.SKU,
				InStock:       p.InStock,
			})
		}
		return products, nil
	}
}

func (a *FirecrawlAdapter) Scrape(ctx context.Context, url string, opts scrapingport.ScrapeOptions) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("firecrawl API key not configured")
	}

	formats := []string{"html"}
	if opts.IncludeMarkdown {
		formats = append(formats, "markdown")
	}

	payload := map[string]interface{}{
		"url":     url,
		"formats": formats,
	}

	respBody, err := a.doRequestWithRetry(ctx, "POST", "/scrape", payload)
	if err != nil {
		return "", fmt.Errorf("firecrawl scrape: %w", err)
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Markdown string `json:"markdown"`
			HTML     string `json:"html"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing scrape response: %w", err)
	}

	if result.Data.Markdown != "" {
		return result.Data.Markdown, nil
	}
	return result.Data.HTML, nil
}

func (a *FirecrawlAdapter) Crawl(ctx context.Context, url string, opts scrapingport.CrawlOptions) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("firecrawl API key not configured")
	}

	payload := map[string]interface{}{
		"url":      url,
		"maxDepth": opts.MaxDepth,
	}
	if len(opts.URLPatterns) > 0 {
		payload["includePaths"] = opts.URLPatterns
	}

	respBody, err := a.doRequestWithRetry(ctx, "POST", "/crawl", payload)
	if err != nil {
		return "", fmt.Errorf("firecrawl crawl: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing crawl response: %w", err)
	}
	return result.ID, nil
}

func (a *FirecrawlAdapter) CrawlStatus(ctx context.Context, jobID string) (scrapingport.CrawlResult, error) {
	if a.apiKey == "" {
		return scrapingport.CrawlResult{}, fmt.Errorf("firecrawl API key not configured")
	}

	respBody, err := a.doRequestWithRetry(ctx, "GET", "/crawl/"+jobID, nil)
	if err != nil {
		return scrapingport.CrawlResult{}, fmt.Errorf("firecrawl crawl status: %w", err)
	}

	var result struct {
		Status    string   `json:"status"`
		Completed int      `json:"completed"`
		Total     int      `json:"total"`
		Data      []string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return scrapingport.CrawlResult{}, fmt.Errorf("parsing crawl status: %w", err)
	}

	return scrapingport.CrawlResult{
		Status:    result.Status,
		Completed: result.Completed,
		Total:     result.Total,
		Data:      result.Data,
	}, nil
}

func (a *FirecrawlAdapter) doRequestWithRetry(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		body, err := a.doRequest(ctx, method, path, payload)
		if err == nil {
			return body, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (a *FirecrawlAdapter) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	var reqBody io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling payload: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("firecrawl API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
