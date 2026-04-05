package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

// TestFirecrawlAdapter_Extract_AsyncPolling verifies regression for 3 bugs:
//   - Bug 1: Extract POSTs to /extract, reads async job id, then polls GET /extract/{id}
//   - Bug 2: data is parsed as a single object (not array)
//   - Bug 3: product "name" field maps to RawProduct.Title
func TestFirecrawlAdapter_Extract_AsyncPolling(t *testing.T) {
	pollCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/extract":
			// Initial submission returns job id (Bug 1: async, not immediate result)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"id":      "job-abc",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/extract/job-abc":
			pollCalls++
			if pollCalls == 1 {
				// First poll: still processing.
				// Real Firecrawl API returns data:[] (empty array) during processing,
				// NOT a struct. The adapter must defer data parsing until status=completed.
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "processing",
					"data":   []interface{}{},
				})
				return
			}
			// Second poll: completed.
			// Bug 2: data is a single object (struct), not an array.
			// Bug 3: product name is in "name" field, not "title".
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "completed",
				"data": map[string]interface{}{
					"products": []map[string]interface{}{
						{
							"name":           "Leche Entera La Serenísima",
							"price":          350.50,
							"original_price": 400.00,
							"url":            "https://example.com/leche",
							"sku":            "LEC-001",
						},
					},
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := newFirecrawlAdapterWithBaseURL("test-api-key", server.URL)

	products, err := adapter.Extract(
		context.Background(),
		"https://example.com/catalog",
		json.RawMessage(`{"type":"object"}`),
		scrapingport.ExtractOptions{},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	p := products[0]

	// Bug 3 regression: name field maps to Title
	if p.Title != "Leche Entera La Serenísima" {
		t.Errorf("Title: expected %q, got %q", "Leche Entera La Serenísima", p.Title)
	}
	if p.Price == nil || *p.Price != 350.50 {
		t.Errorf("Price: expected 350.50, got %v", p.Price)
	}
	if p.SKU != "LEC-001" {
		t.Errorf("SKU: expected %q, got %q", "LEC-001", p.SKU)
	}

	// Bug 1 regression: must have polled at least twice (processing → completed)
	if pollCalls < 2 {
		t.Errorf("expected ≥2 poll calls, got %d (async polling not working)", pollCalls)
	}
}

// TestFirecrawlAdapter_Extract_MissingAPIKey verifies early return when key is absent.
func TestFirecrawlAdapter_Extract_MissingAPIKey(t *testing.T) {
	adapter := newFirecrawlAdapterWithBaseURL("", "http://localhost")
	_, err := adapter.Extract(context.Background(), "https://example.com", nil, scrapingport.ExtractOptions{})
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
}

// TestFirecrawlAdapter_ScrapeJSON_ReturnsStructuredProducts verifies that
// ScrapeJSON calls /v1/scrape with formats:["json"] + jsonOptions and parses
// the structured product list from data.json.products.
func TestFirecrawlAdapter_ScrapeJSON_ReturnsStructuredProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost || r.URL.Path != "/scrape" {
			http.NotFound(w, r)
			return
		}

		// Verify request body includes formats:["json"] and jsonOptions
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		formats, ok := body["formats"].([]interface{})
		if !ok || len(formats) == 0 || formats[0] != "json" {
			http.Error(w, "missing formats:[json]", http.StatusBadRequest)
			return
		}
		if _, ok := body["jsonOptions"]; !ok {
			http.Error(w, "missing jsonOptions", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"json": map[string]interface{}{
					"products": []map[string]interface{}{
						{
							"name":           "Heladera Frío 300L",
							"price":          150000.0,
							"original_price": 180000.0,
							"url":            "https://cetrogar.com.ar/heladera-300",
							"image_url":      "https://cetrogar.com.ar/img/heladera.jpg",
							"brand":          "Drean",
							"category":       "heladera",
							"sku":            "HEL-300",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := newFirecrawlAdapterWithBaseURL("test-api-key", server.URL)
	schema := json.RawMessage(`{"type":"object","properties":{"products":{"type":"array"}}}`)

	products, err := adapter.ScrapeJSON(
		context.Background(),
		"https://cetrogar.com.ar/electrodomesticos.html",
		schema,
		scrapingport.ExtractOptions{Prompt: "Extrae electrodomésticos"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	p := products[0]
	if p.Title != "Heladera Frío 300L" {
		t.Errorf("Title: expected %q, got %q", "Heladera Frío 300L", p.Title)
	}
	if p.Price == nil || *p.Price != 150000.0 {
		t.Errorf("Price: expected 150000.0, got %v", p.Price)
	}
	if p.Brand != "Drean" {
		t.Errorf("Brand: expected %q, got %q", "Drean", p.Brand)
	}
	if p.SKU != "HEL-300" {
		t.Errorf("SKU: expected %q, got %q", "HEL-300", p.SKU)
	}
}

// TestFirecrawlAdapter_ScrapeJSON_EmptyProducts verifies that an empty product
// list in data.json.products returns an empty slice without error.
func TestFirecrawlAdapter_ScrapeJSON_EmptyProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"json": map[string]interface{}{
					"products": []interface{}{},
				},
			},
		})
	}))
	defer server.Close()

	adapter := newFirecrawlAdapterWithBaseURL("test-api-key", server.URL)
	products, err := adapter.ScrapeJSON(context.Background(), "https://example.com", json.RawMessage(`{}`), scrapingport.ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

// TestFirecrawlAdapter_ScrapeJSON_MissingAPIKey verifies early return when key is absent.
func TestFirecrawlAdapter_ScrapeJSON_MissingAPIKey(t *testing.T) {
	adapter := newFirecrawlAdapterWithBaseURL("", "http://localhost")
	_, err := adapter.ScrapeJSON(context.Background(), "https://example.com", nil, scrapingport.ExtractOptions{})
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
}
