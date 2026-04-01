package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
)

type SourceRepository interface {
	Save(ctx context.Context, source *entity.Source) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Source, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, filter SourceFilter) ([]*entity.Source, int, error)
	Update(ctx context.Context, source *entity.Source) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	FindDueForScraping(ctx context.Context) ([]*entity.Source, error)
}

type SourceFilter struct {
	IsActive *bool
	Category string
	Page     int
	PageSize int
}
