package request

type CreateSourceRequest struct {
	Name             string `json:"name"`
	BaseURL          string `json:"base_url"`
	Category         string `json:"category"`
	SourceType       string `json:"source_type"`
	City             string `json:"city"`
	Priority         string `json:"priority"`
	Tier             int    `json:"tier"`
	FirecrawlMethod  string `json:"firecrawl_method"`
	CronExpression   string `json:"cron_expression"`
	Notes            string `json:"notes"`
	ExtractionSchema []byte `json:"extraction_schema"`
	CrawlConfigRaw   []byte `json:"crawl_config"`
}
