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
