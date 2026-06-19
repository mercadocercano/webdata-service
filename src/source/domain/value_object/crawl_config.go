package value_object

import "encoding/json"

// PaginationStrategy determines how the HTTP-JSON adapter paginates.
//   - "range"          — VTEX-style: _from/_to query params, Content-Range total (Cordiez)
//   - "page_increment" — page counter 0,1,2… until empty list (Coop. Obrera)
type PaginationStrategy string

const (
	PaginationRange         PaginationStrategy = "range"
	PaginationPageIncrement PaginationStrategy = "page_increment"
)

type CrawlConfig struct {
	MaxDepth     int      `json:"max_depth,omitempty"`
	URLPatterns  []string `json:"url_patterns,omitempty"`
	MaxPages     int      `json:"max_pages,omitempty"`
	PageParam    string   `json:"page_param,omitempty"`
	IgnoreRobots bool     `json:"ignore_robots,omitempty"`

	// HTTP-JSON adapter extensions (E21 — Coop. Obrera).
	// These fields generalise the adapter beyond VTEX GET+range to support
	// arbitrary JSON APIs (POST body, nested response paths, page_increment).

	// HTTPMethod overrides the request method (default "GET"). Use "POST" for Coop.
	HTTPMethod string `json:"http_method,omitempty"`

	// RequestBodyTemplate is a Go text/template string for POST bodies.
	// Interpolation vars: {{.Page}} (current page 0-based), {{.CategoryID}} (numeric id_busqueda).
	RequestBodyTemplate string `json:"request_body_template,omitempty"`

	// CategoryID is the provider-specific root category identifier passed to
	// RequestBodyTemplate as {{.CategoryID}}. For Coop. Obrera: 2,3,4,5,6,7.
	CategoryID int `json:"category_id,omitempty"`

	// ItemsJSONPath is a dot-separated path to the array of items in the response.
	// Empty string or "." means the response is itself the array (VTEX default).
	// Example: "datos.articulos" for Coop. Obrera.
	ItemsJSONPath string `json:"items_json_path,omitempty"`

	// PaginationStrategy controls how pages are iterated.
	// "range" (default, VTEX) | "page_increment" (Coop. Obrera).
	PaginationStrategy PaginationStrategy `json:"pagination_strategy,omitempty"`
}

func NewCrawlConfig(maxDepth int, urlPatterns []string) CrawlConfig {
	return CrawlConfig{
		MaxDepth:    maxDepth,
		URLPatterns: urlPatterns,
	}
}

func CrawlConfigFromJSON(raw json.RawMessage) (CrawlConfig, error) {
	var cfg CrawlConfig
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return CrawlConfig{}, err
	}
	return cfg, nil
}

func (c CrawlConfig) EffectivePageParam() string {
	if c.PageParam != "" {
		return c.PageParam
	}
	return "page"
}

func (c CrawlConfig) ToJSON() (json.RawMessage, error) {
	return json.Marshal(c)
}
