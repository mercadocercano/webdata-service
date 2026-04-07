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
	SaveBusinessTypes(ctx context.Context, tenantID, productID uuid.UUID, assignments []value_object.BusinessTypeAssignment) error
	RemoveBusinessType(ctx context.Context, tenantID, productID uuid.UUID, code string) error
	FindBusinessTypesForProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.BusinessTypeAssignment, error)
}

// ProductSummary is a lightweight projection used by the auto-match use case.
type ProductSummary struct {
	ID                 uuid.UUID
	Title              string
	Category           string
	NormalizedCategory string
	Brand              string
	BusinessTypes      []value_object.BusinessTypeAssignment
}

type ProductFilter struct {
	SourceID           *uuid.UUID
	Category           string
	NormalizedCategory string
	Brand              string
	BusinessTypeCode   string
	MinPrice           *float64
	MaxPrice           *float64
	Query              string
	SortBy             string
	SortOrder          string
	Page               int
	PageSize           int
	WithoutBusinessTypes bool
}
