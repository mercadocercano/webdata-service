package response

type StatsResponse struct {
	TotalSources   int `json:"total_sources"`
	ActiveSources  int `json:"active_sources"`
	TotalJobs      int `json:"total_jobs"`
	PendingJobs    int `json:"pending_jobs"`
	RunningJobs    int `json:"running_jobs"`
	CompletedJobs  int `json:"completed_jobs"`
	FailedJobs     int `json:"failed_jobs"`
	TotalProducts  int `json:"total_products"`
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
