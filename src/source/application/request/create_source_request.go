package request

import "encoding/json"

type CreateSourceRequest struct {
	Name             string          `json:"name"`
	BaseURL          string          `json:"base_url"`
	Category         string          `json:"category"`
	SourceType       string          `json:"source_type"`
	City             string          `json:"city,omitempty"`
	Priority         string          `json:"priority"`
	Tier             int             `json:"tier"`
	FirecrawlMethod  string          `json:"firecrawl_method,omitempty"`
	CronExpression   string          `json:"cron_expression,omitempty"`
	Notes            string          `json:"notes,omitempty"`
	ExtractionSchema json.RawMessage `json:"extraction_schema,omitempty"`
	CrawlConfig      json.RawMessage `json:"crawl_config,omitempty"`
}
