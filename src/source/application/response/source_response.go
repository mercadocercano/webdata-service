package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
)

type SourceResponse struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	Name               string     `json:"name"`
	BaseURL            string     `json:"base_url"`
	Category           string     `json:"category"`
	SourceType         string     `json:"source_type"`
	City               string     `json:"city,omitempty"`
	Priority           string     `json:"priority"`
	Tier               int        `json:"tier"`
	FirecrawlMethod    string     `json:"firecrawl_method"`
	CronExpression     string     `json:"cron_expression,omitempty"`
	IsActive           bool       `json:"is_active"`
	HealthScore        float64    `json:"health_score"`
	TotalRuns          int        `json:"total_runs"`
	SuccessfulRuns     int        `json:"successful_runs"`
	FailedRuns         int        `json:"failed_runs"`
	LastRunAt          *time.Time `json:"last_run_at,omitempty"`
	LastSuccessAt      *time.Time `json:"last_success_at,omitempty"`
	LastFailureReason  string     `json:"last_failure_reason,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func FromSource(s *entity.Source) SourceResponse {
	return SourceResponse{
		ID:                uuid.UUID(s.ID),
		TenantID:          uuid.UUID(s.TenantID),
		Name:              s.Name,
		BaseURL:           s.BaseURL,
		Category:          s.Category,
		SourceType:        s.SourceType,
		City:              s.City,
		Priority:          s.Priority.Value(),
		Tier:              s.Tier.Value(),
		FirecrawlMethod:   s.FirecrawlMethod,
		CronExpression:    s.CronExpression,
		IsActive:          s.IsActive,
		HealthScore:       s.HealthScore,
		TotalRuns:         s.TotalRuns,
		SuccessfulRuns:    s.SuccessfulRuns,
		FailedRuns:        s.FailedRuns,
		LastRunAt:         s.LastRunAt,
		LastSuccessAt:     s.LastSuccessAt,
		LastFailureReason: s.LastFailureReason,
		Notes:             s.Notes,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}
