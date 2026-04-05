package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type ProductRepository interface {
	Upsert(ctx context.Context, product *entity.ScrapedProduct) (created bool, err error)
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapedProduct, error)
	FindByContentHash(ctx context.Context, tenantID, sourceID uuid.UUID, hash value_object.ContentHash) (*entity.ScrapedProduct, error)
	FindAll(ctx context.Context, tenantID uuid.UUID, filter ProductFilter) ([]*entity.ScrapedProduct, int, error)
	SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error
	BulkSoftDelete(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int64, error)
	UpdateBlocked(ctx context.Context, tenantID, id uuid.UUID, blocked bool) error
	SavePriceRecord(ctx context.Context, record value_object.PriceRecord) error
	FindPriceHistory(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.PriceRecord, error)
}

type ProductFilter struct {
	SourceID           *uuid.UUID
	Category           string
	NormalizedCategory string
	Brand              string
	MinPrice           *float64
	MaxPrice           *float64
	Query              string
	SortBy             string
	SortOrder          string
	Page               int
	PageSize           int
}
