package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type UpsertProductsUseCase struct {
	repo port.ProductRepository
}

func NewUpsertProductsUseCase(repo port.ProductRepository) *UpsertProductsUseCase {
	return &UpsertProductsUseCase{repo: repo}
}

func (uc *UpsertProductsUseCase) Execute(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
) (int, error) {
	created, updated, err := uc.execute(ctx, tenantID, sourceID, jobID, rawProducts)
	return created + updated, err
}

func (uc *UpsertProductsUseCase) ExecuteDetailed(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
) (created, updated int, err error) {
	return uc.execute(ctx, tenantID, sourceID, jobID, rawProducts)
}

func (uc *UpsertProductsUseCase) execute(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
) (created, updated int, err error) {
	for _, raw := range rawProducts {
		if raw.Title == "" || raw.URL == "" {
			continue
		}

		hash := value_object.GenerateContentHash(tenantID, sourceID, raw.Title, raw.URL)

		existing, findErr := uc.repo.FindByContentHash(ctx, tenantID, sourceID, hash)
		if findErr == nil && existing != nil {
			// Product exists — check price change
			if raw.Price != nil && existing.HasPriceChanged(*raw.Price) {
				oldPrice := *existing.Price
				existing.UpdatePrice(*raw.Price)
				if _, saveErr := uc.repo.Upsert(ctx, existing); saveErr == nil {
					record := value_object.NewPriceRecord(tenantID, existing.ID, oldPrice)
					_ = uc.repo.SavePriceRecord(ctx, record)
					updated++
				}
			} else {
				existing.TouchLastSeen()
				if _, saveErr := uc.repo.Upsert(ctx, existing); saveErr == nil {
					updated++
				}
			}
			continue
		}

		// New product
		params := entity.CreateProductParams{
			TenantID:    tenantID,
			SourceID:    sourceID,
			JobID:       jobID,
			Title:       raw.Title,
			Price:       raw.Price,
			OriginalPrice: raw.OriginalPrice,
			URL:         raw.URL,
			ImageURL:    raw.ImageURL,
			Description: raw.Description,
			Brand:       raw.Brand,
			Category:    raw.Category,
			SKU:         raw.SKU,
			EAN:         raw.EAN,
			InStock:     raw.InStock,
			ContentHash: hash,
		}

		product, newErr := entity.NewScrapedProduct(params)
		if newErr != nil {
			continue
		}

		if _, upsertErr := uc.repo.Upsert(ctx, product); upsertErr == nil {
			created++
		}
	}

	return created, updated, nil
}

