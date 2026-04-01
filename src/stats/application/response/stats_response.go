package response

import "github.com/google/uuid"

type StatsResponse struct {
	TotalSources  int     `json:"total_sources"`
	ActiveSources int     `json:"active_sources"`
	TotalProducts int     `json:"total_products"`
	TotalJobs     int     `json:"total_jobs"`
	SuccessRate   float64 `json:"success_rate"`
}

type SourceStatsResponse struct {
	SourceID      uuid.UUID `json:"source_id"`
	Name          string    `json:"name"`
	TotalProducts int       `json:"total_products"`
	TotalJobs     int       `json:"total_jobs"`
	HealthScore   float64   `json:"health_score"`
}
