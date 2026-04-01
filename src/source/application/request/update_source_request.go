package request

import "encoding/json"

type UpdateSourceRequest struct {
	Name             *string         `json:"name,omitempty"`
	BaseURL          *string         `json:"base_url,omitempty"`
	Category         *string         `json:"category,omitempty"`
	City             *string         `json:"city,omitempty"`
	Priority         *string         `json:"priority,omitempty"`
	Tier             *int            `json:"tier,omitempty"`
	FirecrawlMethod  *string         `json:"firecrawl_method,omitempty"`
	CronExpression   *string         `json:"cron_expression,omitempty"`
	Notes            *string         `json:"notes,omitempty"`
	IsActive         *bool           `json:"is_active,omitempty"`
	ExtractionSchema json.RawMessage `json:"extraction_schema,omitempty"`
	CrawlConfig      json.RawMessage `json:"crawl_config,omitempty"`
}
