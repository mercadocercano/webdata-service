package port

import (
	"context"
	"encoding/json"
)

type ScraperPort interface {
	Extract(ctx context.Context, url string, schema json.RawMessage, opts ExtractOptions) ([]RawProduct, error)
	ScrapeJSON(ctx context.Context, url string, schema json.RawMessage, opts ExtractOptions) ([]RawProduct, error)
	Scrape(ctx context.Context, url string, opts ScrapeOptions) (string, error)
	Crawl(ctx context.Context, url string, opts CrawlOptions) (string, error)
	CrawlStatus(ctx context.Context, jobID string) (CrawlResult, error)
	// FetchHTTPJSON fetches products directly from a JSON HTTP API without any
	// headless browser or LLM. Currently used for VTEX storefronts (e.g. Cordiez)
	// via firecrawl_method='http_json'. The opts carry provider-specific config
	// (page size, max products). Designed to be generalized later for other
	// JSON-native APIs (Magento, Cooperativa Obrera, etc.).
	FetchHTTPJSON(ctx context.Context, baseURL string, opts HTTPJSONOptions) ([]RawProduct, error)
}

// HTTPJSONOptions carries provider-specific config for FetchHTTPJSON.
// Fields are populated from source.CrawlConfig by the use case before calling the adapter.
type HTTPJSONOptions struct {
	// PageSize is the number of items per HTTP request (default 50, VTEX max is 50).
	PageSize int
	// MaxProducts caps the total products fetched across all pages (0 = unlimited).
	MaxProducts int

	// --- HTTP-JSON generalisation fields (E21 — Coop. Obrera) ---

	// HTTPMethod overrides the request method. Default "GET" (VTEX). Use "POST" for Coop.
	HTTPMethod string

	// RequestBodyTemplate is a Go text/template for POST bodies.
	// Template vars: {{.Page}} int, {{.CategoryID}} int.
	RequestBodyTemplate string

	// CategoryID is passed to RequestBodyTemplate as {{.CategoryID}}.
	CategoryID int

	// ItemsJSONPath is a dot-separated path to the items array in the response.
	// Empty means the response root IS the array (VTEX). Example: "datos.articulos".
	ItemsJSONPath string

	// PaginationStrategy selects how pages are iterated:
	//   ""/"range"         — VTEX _from/_to (default)
	//   "page_increment"   — page counter 0,1,2… until empty list (Coop. Obrera)
	PaginationStrategy string
}

type RawProduct struct {
	Title         string
	Price         *float64
	OriginalPrice *float64
	URL           string
	ImageURL      string
	Description   string
	Brand         string
	Category      string
	SKU           string
	EAN           string
	InStock       *bool
}

type ExtractOptions struct {
	Prompt string
}

type ScrapeOptions struct {
	IncludeMarkdown bool
}

type CrawlOptions struct {
	MaxDepth    int
	URLPatterns []string
}

type CrawlResult struct {
	Status    string
	Completed int
	Total     int
	Data      []string
}
