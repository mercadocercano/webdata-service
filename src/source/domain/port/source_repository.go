package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
)

type SourceRepository interface {
	Save(ctx context.Context, source *entity.Source) error
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Source, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, filter SourceFilter) ([]*entity.Source, int, error)
	// FindAllActiveByCategoryPlatform returns the active sources for a category across ALL
	// tenants. It is a PLATFORM read path, not a tenant-scoped one: there is no `tenantID`
	// parameter and no RLS backstop on `webdata_sources`, so this method MUST be invoked
	// only under verified platform auth (system_admin) — never from a tenant handler —
	// exactly the bypass cross-tenant that PLAT-E39 exists to close (D9a, condición O2).
	// Tenant-scoped reads use FindAll(ctx, tenantID, filter). The only allowed caller is the
	// enrichment usecase (src/product/application/usecase); a guard test enforces that.
	FindAllActiveByCategoryPlatform(ctx context.Context, category string) ([]*entity.Source, error)
	Update(ctx context.Context, source *entity.Source) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	FindDueForScraping(ctx context.Context) ([]*entity.Source, error)
	UpdateNextRunAt(ctx context.Context, sourceID uuid.UUID, nextRunAt time.Time) error
	RecordJobResult(ctx context.Context, sourceID uuid.UUID, success bool) error
}

type SourceFilter struct {
	IsActive *bool
	Category string
	Page     int
	PageSize int
}
