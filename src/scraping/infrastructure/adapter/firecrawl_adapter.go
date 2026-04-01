package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

const (
	defaultBaseURL    = "https://api.firecrawl.dev/v1"
	maxRetries        = 3
	initialRetryDelay = 1 * time.Second
)

type FirecrawlAdapter struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewFirecrawlAdapter(apiKey string) *FirecrawlAdapter {
	return &FirecrawlAdapter{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
	}
}

func (a *FirecrawlAdapter) Extract(ctx context.Context, url string, schema json.RawMessage, opts port.ExtractOptions) ([]port.RawProduct, error) {
	payload := map[string]interface{}{
		"urls":   []string{url},
		"schema": schema,
	}
	if opts.Prompt != "" {
		payload["prompt"] = opts.Prompt
	}

	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			Title         string   `json:"title"`
			Price         *float64 `json:"price"`
			OriginalPrice *float64 `json:"original_price"`
			URL           string   `json:"url"`
			ImageURL      string   `json:"image_url"`
			Description   string   `json:"description"`
			Brand         string   `json:"brand"`
			Category      string   `json:"category"`
			SKU           string   `json:"sku"`
			EAN           string   `json:"ean"`
			InStock       *bool    `json:"in_stock"`
		} `json:"data"`
	}

	if err := a.postWithRetry(ctx, "/extract", payload, &resp); err != nil {
		return nil, fmt.Errorf("firecrawl extract: %w", err)
	}

	products := make([]port.RawProduct, 0, len(resp.Data))
	for _, d := range resp.Data {
		products = append(products, port.RawProduct{
			Title:         d.Title,
			Price:         d.Price,
			OriginalPrice: d.OriginalPrice,
			URL:           d.URL,
			ImageURL:      d.ImageURL,
			Description:   d.Description,
			Brand:         d.Brand,
			Category:      d.Category,
			SKU:           d.SKU,
			EAN:           d.EAN,
			InStock:       d.InStock,
		})
	}
	return products, nil
}

func (a *FirecrawlAdapter) Scrape(ctx context.Context, url string, opts port.ScrapeOptions) (string, error) {
	payload := map[string]interface{}{
		"url": url,
		"formats": func() []string {
			if opts.IncludeMarkdown {
				return []string{"markdown"}
			}
			return []string{"html"}
		}(),
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Markdown string `json:"markdown"`
			HTML     string `json:"html"`
		} `json:"data"`
	}

	if err := a.postWithRetry(ctx, "/scrape", payload, &resp); err != nil {
		return "", fmt.Errorf("firecrawl scrape: %w", err)
	}

	if opts.IncludeMarkdown {
		return resp.Data.Markdown, nil
	}
	return resp.Data.HTML, nil
}

func (a *FirecrawlAdapter) Crawl(ctx context.Context, url string, opts port.CrawlOptions) (string, error) {
	payload := map[string]interface{}{
		"url":      url,
		"maxDepth": opts.MaxDepth,
	}
	if len(opts.URLPatterns) > 0 {
		payload["includePaths"] = opts.URLPatterns
	}

	var resp struct {
		Success bool   `json:"success"`
		ID      string `json:"id"`
	}

	if err := a.postWithRetry(ctx, "/crawl", payload, &resp); err != nil {
		return "", fmt.Errorf("firecrawl crawl: %w", err)
	}
	return resp.ID, nil
}

func (a *FirecrawlAdapter) CrawlStatus(ctx context.Context, jobID string) (port.CrawlResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/crawl/"+jobID, nil)
	if err != nil {
		return port.CrawlResult{}, fmt.Errorf("building crawl status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	httpResp, err := a.httpClient.Do(req)
	if err != nil {
		return port.CrawlResult{}, fmt.Errorf("firecrawl crawl status: %w", err)
	}
	defer httpResp.Body.Close()

	var resp struct {
		Status    string `json:"status"`
		Completed int    `json:"completed"`
		Total     int    `json:"total"`
		Data      []struct {
			Markdown string `json:"markdown"`
		} `json:"data"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return port.CrawlResult{}, fmt.Errorf("decoding crawl status: %w", err)
	}

	data := make([]string, len(resp.Data))
	for i, d := range resp.Data {
		data[i] = d.Markdown
	}

	return port.CrawlResult{
		Status:    resp.Status,
		Completed: resp.Completed,
		Total:     resp.Total,
		Data:      data,
	}, nil
}

func (a *FirecrawlAdapter) postWithRetry(ctx context.Context, path string, payload interface{}, result interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * initialRetryDelay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("client error %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
		return nil
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
