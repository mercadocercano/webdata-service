package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
)

type SourceResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Name             string     `json:"name"`
	BaseURL          string     `json:"base_url"`
	Category         string     `json:"category"`
	SourceType       string     `json:"source_type"`
	City             string     `json:"city"`
	Priority         string     `json:"priority"`
	Tier             int        `json:"tier"`
	FirecrawlMethod  string     `json:"firecrawl_method"`
	CronExpression   string     `json:"cron_expression"`
	IsActive         bool       `json:"is_active"`
	HealthScore      float64    `json:"health_score"`
	TotalRuns        int        `json:"total_runs"`
	SuccessfulRuns   int        `json:"successful_runs"`
	FailedRuns       int        `json:"failed_runs"`
	Notes            string     `json:"notes"`
	NextRunAt        *time.Time `json:"next_run_at"`
	LastRunAt        *time.Time `json:"last_run_at"`
	LastSuccessAt    *time.Time `json:"last_success_at"`
	LastFailureReason string    `json:"last_failure_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func FromSource(s *entity.Source) SourceResponse {
	return SourceResponse{
		ID:                s.ID,
		TenantID:          s.TenantID,
		Name:              s.Name,
		BaseURL:           s.BaseURL,
		Category:          s.Category,
		SourceType:        s.SourceType,
		City:              s.City,
		Priority:          s.Priority.String(),
		Tier:              s.Tier.Value(),
		FirecrawlMethod:   s.FirecrawlMethod,
		CronExpression:    s.CronExpression,
		IsActive:          s.IsActive,
		HealthScore:       s.HealthScore,
		TotalRuns:         s.TotalRuns,
		SuccessfulRuns:    s.SuccessfulRuns,
		FailedRuns:        s.FailedRuns,
		Notes:             s.Notes,
		NextRunAt:         s.NextRunAt,
		LastRunAt:         s.LastRunAt,
		LastSuccessAt:     s.LastSuccessAt,
		LastFailureReason: s.LastFailureReason,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}
