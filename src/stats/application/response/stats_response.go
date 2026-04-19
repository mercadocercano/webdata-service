package response

import "time"

type StatsResponse struct {
	TotalSources  int `json:"total_sources"`
	ActiveSources int `json:"active_sources"`
	TotalJobs     int `json:"total_jobs"`
	PendingJobs   int `json:"pending_jobs"`
	RunningJobs   int `json:"running_jobs"`
	CompletedJobs int `json:"completed_jobs"`
	FailedJobs    int `json:"failed_jobs"`
	TotalProducts int `json:"total_products"`
}

type SourceStatsResponse struct {
	SourceID       string  `json:"source_id"`
	Name           string  `json:"name"`
	HealthScore    float64 `json:"health_score"`
	TotalRuns      int     `json:"total_runs"`
	SuccessfulRuns int     `json:"successful_runs"`
	FailedRuns     int     `json:"failed_runs"`
	IsActive       bool    `json:"is_active"`
}

type PipelineStatusResponse struct {
	BusinessType string          `json:"business_type"`
	Sources      SourcesSummary  `json:"sources"`
	Products     ProductsSummary `json:"products"`
	LastScrape   *time.Time      `json:"last_scrape"`
	NextScrape   *time.Time      `json:"next_scrape"`
}

type SourcesSummary struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Healthy int `json:"healthy"`
	Failing int `json:"failing"`
}

type ProductsSummary struct {
	Scraped      int `json:"scraped"`
	WithImage    int `json:"with_image"`
	WithoutImage int `json:"without_image"`
	SyncedToPIM  int `json:"synced_to_pim"`
	PendingSync  int `json:"pending_sync"`
}
