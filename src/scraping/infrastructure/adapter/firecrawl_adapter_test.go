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
				// First poll: still processing
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "processing"})
				return
			}
			// Second poll: completed
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
