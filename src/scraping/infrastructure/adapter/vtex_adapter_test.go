package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

// buildVTEXBody serializes a slice of product maps into the JSON array body
// that the VTEX catalog_system/pub/products/search API returns.
func buildVTEXBody(products []map[string]interface{}) []byte {
	b, _ := json.Marshal(products)
	return b
}

// vtexProductPayload returns a minimal but complete VTEX product map with
// all fields that mapVTEXProduct reads.
func vtexProductPayload(name, brand, ean, sku, imageURL, directLink, linkText, category string, price, listPrice float64, inStock bool) map[string]interface{} {
	return map[string]interface{}{
		"productName": name,
		"brand":       brand,
		"link":        directLink,
		"linkText":    linkText,
		"categories":  []string{category},
		"items": []map[string]interface{}{
			{
				"itemId": sku,
				"ean":    ean,
				"images": []map[string]interface{}{
					{"imageUrl": imageURL},
				},
				"sellers": []map[string]interface{}{
					{
						"commertialOffer": map[string]interface{}{
							"Price":       price,
							"ListPrice":   listPrice,
							"IsAvailable": inStock,
						},
					},
				},
			},
		},
	}
}

// TestVTEXAdapter_FetchHTTPJSON_MapsAllFields verifies that every RawProduct
// field is populated correctly from a real VTEX API response shape.
func TestVTEXAdapter_FetchHTTPJSON_MapsAllFields(t *testing.T) {
	payload := vtexProductPayload(
		"Leche Entera La Serenísima 1L",
		"La Serenísima",
		"7790070011234",
		"987654",
		"https://arcordiezb2c.vteximg.com.br/leche.jpg",
		"https://www.cordiez.com.ar/leche-entera-la-serenisima/p",
		"leche-entera-la-serenisima",
		"/Lácteos/Leches/",
		1250.50,
		1500.00,
		true,
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Range", "items 0-0/1")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(buildVTEXBody([]map[string]interface{}{payload}))
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/api/catalog_system/pub/products/search/",
		scrapingport.HTTPJSONOptions{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	p := products[0]

	if p.Title != "Leche Entera La Serenísima 1L" {
		t.Errorf("Title: expected %q, got %q", "Leche Entera La Serenísima 1L", p.Title)
	}
	if p.Brand != "La Serenísima" {
		t.Errorf("Brand: expected %q, got %q", "La Serenísima", p.Brand)
	}
	if p.EAN != "7790070011234" {
		t.Errorf("EAN: expected %q, got %q", "7790070011234", p.EAN)
	}
	if p.SKU != "987654" {
		t.Errorf("SKU: expected %q, got %q", "987654", p.SKU)
	}
	if p.ImageURL != "https://arcordiezb2c.vteximg.com.br/leche.jpg" {
		t.Errorf("ImageURL: expected VTEX CDN URL, got %q", p.ImageURL)
	}
	if p.Price == nil || *p.Price != 1250.50 {
		t.Errorf("Price: expected 1250.50, got %v", p.Price)
	}
	if p.OriginalPrice == nil || *p.OriginalPrice != 1500.00 {
		t.Errorf("OriginalPrice: expected 1500.00, got %v", p.OriginalPrice)
	}
	if p.InStock == nil || !*p.InStock {
		t.Errorf("InStock: expected true, got %v", p.InStock)
	}
	if p.Category != "/Lácteos/Leches/" {
		t.Errorf("Category: expected %q, got %q", "/Lácteos/Leches/", p.Category)
	}
	if p.URL != "https://www.cordiez.com.ar/leche-entera-la-serenisima/p" {
		t.Errorf("URL: expected direct link, got %q", p.URL)
	}
}

// TestVTEXAdapter_FetchHTTPJSON_HTTP206IsSuccess verifies that HTTP 206
// (Partial Content) is treated as a successful response.
// VTEX always returns 206 for paginated catalog requests.
func TestVTEXAdapter_FetchHTTPJSON_HTTP206IsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Range", "items 0-0/1")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(buildVTEXBody([]map[string]interface{}{
			{"productName": "Test", "items": []map[string]interface{}{{"itemId": "1", "ean": "", "images": []interface{}{}, "sellers": []interface{}{}}}},
		}))
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(context.Background(), server.URL+"/search/", scrapingport.HTTPJSONOptions{})
	if err != nil {
		t.Fatalf("HTTP 206 must be treated as success, got error: %v", err)
	}
	if len(products) != 1 {
		t.Errorf("expected 1 product from 206 response, got %d", len(products))
	}
}

// TestVTEXAdapter_FetchHTTPJSON_Pagination verifies range-based pagination:
// the adapter should make multiple requests and aggregate all products.
// Simulates: total=3, page_size=2 → 2 requests.
func TestVTEXAdapter_FetchHTTPJSON_Pagination(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		from := r.URL.Query().Get("_from")
		w.Header().Set("Content-Range", "items 0-2/3")
		w.WriteHeader(http.StatusPartialContent)

		switch from {
		case "0":
			w.Write(buildVTEXBody([]map[string]interface{}{
				{"productName": "Producto 1", "items": []map[string]interface{}{{"itemId": "1", "ean": "E1", "images": []interface{}{}, "sellers": []interface{}{}}}},
				{"productName": "Producto 2", "items": []map[string]interface{}{{"itemId": "2", "ean": "E2", "images": []interface{}{}, "sellers": []interface{}{}}}},
			}))
		case "2":
			w.Write(buildVTEXBody([]map[string]interface{}{
				{"productName": "Producto 3", "items": []map[string]interface{}{{"itemId": "3", "ean": "E3", "images": []interface{}{}, "sellers": []interface{}{}}}},
			}))
		default:
			// Empty page — should stop pagination
			w.Write(buildVTEXBody([]map[string]interface{}{}))
		}
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/search/",
		scrapingport.HTTPJSONOptions{PageSize: 2},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 3 {
		t.Errorf("expected 3 products across 2 pages, got %d", len(products))
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", callCount)
	}

	wantTitles := []string{"Producto 1", "Producto 2", "Producto 3"}
	for i, want := range wantTitles {
		if products[i].Title != want {
			t.Errorf("products[%d].Title: expected %q, got %q", i, want, products[i].Title)
		}
	}
}

// TestVTEXAdapter_FetchHTTPJSON_MaxProducts verifies that MaxProducts caps
// the total products fetched and stops pagination early.
func TestVTEXAdapter_FetchHTTPJSON_MaxProducts(t *testing.T) {
	callCount := 0

	// Returns 50 items per page, total=409
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Range", "items 0-49/409")
		w.WriteHeader(http.StatusPartialContent)

		products := make([]map[string]interface{}, 50)
		for i := range products {
			products[i] = map[string]interface{}{
				"productName": "Producto",
				"items":       []map[string]interface{}{{"itemId": "x", "ean": "", "images": []interface{}{}, "sellers": []interface{}{}}},
			}
		}
		w.Write(buildVTEXBody(products))
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/search/",
		scrapingport.HTTPJSONOptions{PageSize: 50, MaxProducts: 50},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 50 {
		t.Errorf("expected 50 products (MaxProducts cap), got %d", len(products))
	}
	if callCount != 1 {
		t.Errorf("expected 1 request (MaxProducts=50, page_size=50), got %d", callCount)
	}
}

// TestVTEXAdapter_FetchHTTPJSON_URLFallbackFromLinkText verifies that when
// the VTEX "link" field is empty, the URL is constructed from linkText + host.
func TestVTEXAdapter_FetchHTTPJSON_URLFallbackFromLinkText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Range", "items 0-0/1")
		w.WriteHeader(http.StatusPartialContent)
		w.Write(buildVTEXBody([]map[string]interface{}{
			{
				"productName": "Azúcar Ledesma 1kg",
				"link":        "", // empty — must fall back to linkText
				"linkText":    "azucar-ledesma-1kg",
				"items":       []map[string]interface{}{{"itemId": "55", "ean": "EAN55", "images": []interface{}{}, "sellers": []interface{}{}}},
			},
		}))
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/api/catalog_system/pub/products/search/",
		scrapingport.HTTPJSONOptions{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].URL == "" {
		t.Error("URL should be constructed from linkText when link is empty")
	}
	// URL should contain the slug.
	parsedURL, _ := url.Parse(products[0].URL)
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		t.Errorf("URL should contain slug path, got %q", products[0].URL)
	}
}

// TestVTEXAdapter_FetchHTTPJSON_VTEXOffsetCap verifies that the adapter stops
// pagination at _from=2500 without returning an error. The VTEX public API
// rejects _from > 2500 with HTTP 400 ("Parameter _from can't be greater than
// 2500."). Categories with >2500 products (e.g. Carrefour Almacén=6101) must
// be seeded as subcategories to get full coverage.
func TestVTEXAdapter_FetchHTTPJSON_VTEXOffsetCap(t *testing.T) {
	callCount := 0
	pageSize := 50

	// Simulate a category with total > 2500. The adapter should stop at 2500
	// and not attempt _from=2500 (which would return HTTP 400 from VTEX).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		from, _ := strconv.Atoi(r.URL.Query().Get("_from"))

		// If the adapter correctly caps at 2500, this should never be called
		// with from >= 2500.
		if from >= 2500 {
			// Simulate VTEX HTTP 400 for _from > 2500
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`"Parameter _from can't be greater than 2500."`))
			return
		}

		// Report total=6000 (> vtexMaxOffset=2500)
		w.Header().Set("Resources", fmt.Sprintf("%d-%d/6000", from, from+pageSize-1))
		w.WriteHeader(http.StatusPartialContent)

		prods := make([]map[string]interface{}, pageSize)
		for i := range prods {
			prods[i] = map[string]interface{}{
				"productName": fmt.Sprintf("Prod %d", from+i),
				"items":       []map[string]interface{}{{"itemId": fmt.Sprintf("%d", from+i), "ean": fmt.Sprintf("EAN%d", from+i), "images": []interface{}{}, "sellers": []interface{}{}}},
			}
		}
		w.Write(buildVTEXBody(prods))
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/search/",
		scrapingport.HTTPJSONOptions{PageSize: pageSize},
	)
	if err != nil {
		t.Fatalf("expected no error (cap should be graceful), got: %v", err)
	}

	// Should have fetched exactly vtexMaxOffset=2500 products (50 pages × 50 prods)
	if len(products) != vtexMaxOffset {
		t.Errorf("expected %d products (vtexMaxOffset cap), got %d", vtexMaxOffset, len(products))
	}

	// Should have made exactly 2500/50=50 requests, all with _from < 2500
	expectedCalls := vtexMaxOffset / pageSize
	if callCount != expectedCalls {
		t.Errorf("expected %d HTTP requests (no call with _from >= 2500), got %d", expectedCalls, callCount)
	}
}

// TestVTEXAdapter_FetchHTTPJSON_ResourcesHeader verifies that the adapter
// reads the "Resources" header (used by Día, Carrefour, La Anónima, Farmacity)
// in addition to "Content-Range" (used by Cordiez). Both storefronts run VTEX
// but send different headers for the pagination total.
func TestVTEXAdapter_FetchHTTPJSON_ResourcesHeader(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		from := r.URL.Query().Get("_from")
		// Simulate Día/Carrefour style: Resources header, NOT Content-Range.
		w.Header().Set("Resources", "0-2/3")
		w.WriteHeader(http.StatusPartialContent)

		switch from {
		case "0":
			w.Write(buildVTEXBody([]map[string]interface{}{
				{"productName": "P1", "items": []map[string]interface{}{{"itemId": "1", "ean": "E1", "images": []interface{}{}, "sellers": []interface{}{}}}},
				{"productName": "P2", "items": []map[string]interface{}{{"itemId": "2", "ean": "E2", "images": []interface{}{}, "sellers": []interface{}{}}}},
			}))
		case "2":
			w.Write(buildVTEXBody([]map[string]interface{}{
				{"productName": "P3", "items": []map[string]interface{}{{"itemId": "3", "ean": "E3", "images": []interface{}{}, "sellers": []interface{}{}}}},
			}))
		default:
			w.Write(buildVTEXBody([]map[string]interface{}{}))
		}
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	products, err := a.FetchHTTPJSON(
		context.Background(),
		server.URL+"/search/",
		scrapingport.HTTPJSONOptions{PageSize: 2},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 3 {
		t.Errorf("expected 3 products, got %d", len(products))
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP requests (Resources header pagination), got %d", callCount)
	}
}

// TestVTEXAdapter_FetchHTTPJSON_HTTP4xxReturnsError verifies that non-2xx/206
// responses are treated as errors.
func TestVTEXAdapter_FetchHTTPJSON_HTTP4xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	a := newVTEXAdapterWithClient(server.Client())
	_, err := a.FetchHTTPJSON(context.Background(), server.URL+"/search/", scrapingport.HTTPJSONOptions{})
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
}

// TestParseContentRangeTotal_Table verifies Content-Range and Resources header
// parsing for all expected VTEX response formats.
// Note: Cordiez uses "Content-Range: items 0-49/409"; Día/Carrefour/La Anónima/
// Farmacity use "Resources: 0-49/409". Both are parsed by parseContentRangeTotal.
func TestParseContentRangeTotal_Table(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"vtex content-range format", "items 0-49/409", 409},
		{"vtex resources format (Día/Carrefour/MasOnline/Farmacity)", "0-49/409", 409},
		{"single item", "items 0-0/1", 1},
		{"resources single item", "0-0/1", 1},
		{"empty header", "", -1},
		{"no slash", "items 0-49", -1},
		{"empty total", "items 0-49/", -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseContentRangeTotal(tc.header)
			if got != tc.want {
				t.Errorf("parseContentRangeTotal(%q): expected %d, got %d", tc.header, tc.want, got)
			}
		})
	}
}

// TestBuildVTEXPageURL_Table verifies that range params are injected into
// VTEX URLs correctly, including URLs with pre-existing query params (e.g. fq=).
func TestBuildVTEXPageURL_Table(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		from        int
		to          int
		wantFrom    string
		wantTo      string
		wantExtraFQ string // non-empty means fq param must be present
	}{
		{
			name:    "plain URL",
			baseURL: "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/",
			from:    0, to: 49,
			wantFrom: "0", wantTo: "49",
		},
		{
			name:        "URL with category filter",
			baseURL:     "https://www.cordiez.com.ar/api/catalog_system/pub/products/search/?fq=C:/10003/",
			from:        50, to: 99,
			wantFrom:    "50", wantTo: "99",
			wantExtraFQ: "C:/10003/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildVTEXPageURL(tc.baseURL, tc.from, tc.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("result is not a valid URL: %v", err)
			}
			q := u.Query()
			if q.Get("_from") != tc.wantFrom {
				t.Errorf("_from: expected %q, got %q", tc.wantFrom, q.Get("_from"))
			}
			if q.Get("_to") != tc.wantTo {
				t.Errorf("_to: expected %q, got %q", tc.wantTo, q.Get("_to"))
			}
			if tc.wantExtraFQ != "" && q.Get("fq") != tc.wantExtraFQ {
				t.Errorf("fq: expected %q, got %q", tc.wantExtraFQ, q.Get("fq"))
			}
		})
	}
}
