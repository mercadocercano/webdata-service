package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
)

type JobResponse struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	SourceID       uuid.UUID  `json:"source_id"`
	Status         string     `json:"status"`
	TriggerType    string     `json:"trigger_type"`
	FirecrawlJobID string     `json:"firecrawl_job_id,omitempty"`
	ProductsFound  int        `json:"products_found"`
	ProductsSaved  int        `json:"products_saved"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	RetryCount     int        `json:"retry_count"`
	MaxRetries     int        `json:"max_retries"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func FromJob(j *entity.ScrapingJob) JobResponse {
	return JobResponse{
		ID:             j.ID,
		TenantID:       j.TenantID,
		SourceID:       j.SourceID,
		Status:         j.Status.Value(),
		TriggerType:    j.TriggerType.Value(),
		FirecrawlJobID: j.FirecrawlJobID,
		ProductsFound:  j.ProductsFound,
		ProductsSaved:  j.ProductsSaved,
		ErrorMessage:   j.ErrorMessage,
		RetryCount:     j.RetryCount,
		MaxRetries:     j.MaxRetries,
		StartedAt:      j.StartedAt,
		CompletedAt:    j.CompletedAt,
		CreatedAt:      j.CreatedAt,
	}
}
