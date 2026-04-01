package request

type UpdateSourceRequest struct {
	Name             *string `json:"name"`
	BaseURL          *string `json:"base_url"`
	Category         *string `json:"category"`
	City             *string `json:"city"`
	Priority         *string `json:"priority"`
	FirecrawlMethod  *string `json:"firecrawl_method"`
	CronExpression   *string `json:"cron_expression"`
	Notes            *string `json:"notes"`
	IsActive         *bool   `json:"is_active"`
	ExtractionSchema []byte  `json:"extraction_schema"`
	CrawlConfigRaw   []byte  `json:"crawl_config"`
}
