package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
)

type ScrapingJobRepository interface {
	Save(ctx context.Context, job *entity.ScrapingJob) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapingJob, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, filter JobFilter) ([]*entity.ScrapingJob, int, error)
	Update(ctx context.Context, job *entity.ScrapingJob) error
	ClaimPendingJob(ctx context.Context) (*entity.ScrapingJob, error)
}

type JobFilter struct {
	SourceID *uuid.UUID
	Status   string
	Page     int
	PageSize int
}
