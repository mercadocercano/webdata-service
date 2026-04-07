package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type UpsertProductsUseCase struct {
	repo           port.ProductRepository
	categoryFinder port.SourceCategoryFinder
	pimSync        *SyncProductToPIMUseCase
	sourceName     string
}

func NewUpsertProductsUseCase(repo port.ProductRepository, categoryFinder port.SourceCategoryFinder) *UpsertProductsUseCase {
	return &UpsertProductsUseCase{repo: repo, categoryFinder: categoryFinder}
}

func (uc *UpsertProductsUseCase) WithPIMSync(syncUC *SyncProductToPIMUseCase) {
	uc.pimSync = syncUC
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
	// Resolve source category once for the entire batch.
	var autoAssignment *value_object.BusinessTypeAssignment
	if uc.categoryFinder != nil {
		if category, catErr := uc.categoryFinder.FindCategoryBySourceID(ctx, tenantID, sourceID); catErr == nil {
			if assignment, ok := value_object.MapCategoryToBusinessType(category); ok {
				autoAssignment = &assignment
			}
		}
	}

	for _, raw := range rawProducts {
		if raw.Title == "" {
			continue
		}

		hash := value_object.GenerateContentHash(tenantID, sourceID, raw.Title, raw.URL)

		existing, findErr := uc.repo.FindByContentHash(ctx, tenantID, sourceID, hash)
		if findErr == nil && existing != nil {
			// Enrich missing fields first (image, category, brand, etc.)
			existing.EnrichFields(raw.ImageURL, raw.Category, raw.Brand, raw.Description, raw.URL, raw.Price)

			// Check price change to record history
			if raw.Price != nil && existing.Price != nil && existing.HasPriceChanged(*raw.Price) {
				oldPrice := *existing.Price
				existing.UpdatePrice(*raw.Price)
				if _, saveErr := uc.repo.Upsert(ctx, existing); saveErr == nil {
					record := value_object.NewPriceRecord(tenantID, existing.ID, oldPrice)
					_ = uc.repo.SavePriceRecord(ctx, record)
					updated++
				} else {
					fmt.Printf("[upsert] error saving price update for product %s (title=%q): %v\n", existing.ID, existing.Title, saveErr)
				}
			} else {
				existing.TouchLastSeen()
				if _, saveErr := uc.repo.Upsert(ctx, existing); saveErr == nil {
					updated++
				} else {
					fmt.Printf("[upsert] error touching last_seen for product %s (title=%q): %v\n", existing.ID, existing.Title, saveErr)
				}
			}
			uc.trySyncToPIM(ctx, existing)
			continue
		}

		// New product
		params := entity.CreateProductParams{
			TenantID:      tenantID,
			SourceID:      sourceID,
			JobID:         jobID,
			Title:         raw.Title,
			Price:         raw.Price,
			OriginalPrice: raw.OriginalPrice,
			URL:           raw.URL,
			ImageURL:      raw.ImageURL,
			Description:   raw.Description,
			Brand:         raw.Brand,
			Category:      raw.Category,
			SKU:           raw.SKU,
			EAN:           raw.EAN,
			InStock:       raw.InStock,
			ContentHash:   hash,
		}

		product, newErr := entity.NewScrapedProduct(params)
		if newErr != nil {
			fmt.Printf("[upsert] error constructing product (title=%q): %v\n", raw.Title, newErr)
			continue
		}

		if _, upsertErr := uc.repo.Upsert(ctx, product); upsertErr == nil {
			created++
			if autoAssignment != nil {
				assignments := []value_object.BusinessTypeAssignment{*autoAssignment}
				if btErr := uc.repo.SaveBusinessTypes(ctx, tenantID, product.ID, assignments); btErr != nil {
					fmt.Printf("[upsert] error auto-assigning business type for product %s: %v\n", product.ID, btErr)
				}
				product.BusinessTypes = assignments
			}
			uc.trySyncToPIM(ctx, product)
		} else {
			fmt.Printf("[upsert] error saving new product (title=%q, hash=%s): %v\n", product.Title, product.ContentHash, upsertErr)
		}
	}

	return created, updated, nil
}

func (uc *UpsertProductsUseCase) trySyncToPIM(ctx context.Context, product *entity.ScrapedProduct) {
	if uc.pimSync == nil || !product.NeedsPIMSync() {
		return
	}
	if err := uc.pimSync.Execute(ctx, product, uc.sourceName); err != nil {
		log.Printf("[upsert] pim-sync failed for product %s: %v", product.ID, err)
	}
}

