package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	webdataport "github.com/mercadocercano/webdata-service/src/webdata/domain/port"
)

type UpsertProductsUseCase struct {
	repo           port.ProductRepository
	categoryFinder port.SourceCategoryFinder
	pimSync        *SyncProductToPIMUseCase
	sourceName     string
	logger         webdataport.WebdataEventLogger
}

func NewUpsertProductsUseCase(repo port.ProductRepository, categoryFinder port.SourceCategoryFinder) *UpsertProductsUseCase {
	return &UpsertProductsUseCase{repo: repo, categoryFinder: categoryFinder}
}

// WithLogger inyecta el logger canónico (ADR-001). Nil-safe: si no se llama, los eventos se descartan.
func (uc *UpsertProductsUseCase) WithLogger(logger webdataport.WebdataEventLogger) {
	uc.logger = logger
}

func (uc *UpsertProductsUseCase) log(e webdataport.WebdataEvent) {
	if uc.logger != nil {
		uc.logger.Log(e)
	}
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
	created, updated, err := uc.execute(ctx, tenantID, sourceID, jobID, rawProducts, nil)
	return created + updated, err
}

func (uc *UpsertProductsUseCase) ExecuteDetailed(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
) (created, updated int, err error) {
	return uc.execute(ctx, tenantID, sourceID, jobID, rawProducts, nil)
}

// ExecuteDetailedWithFilter aplica el filtro de marcas excluidas antes de persistir.
// excludedBrands es el valor de source.ExcludedBrands resuelto por el caller.
func (uc *UpsertProductsUseCase) ExecuteDetailedWithFilter(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
	excludedBrands []string,
) (created, updated int, err error) {
	return uc.execute(ctx, tenantID, sourceID, jobID, rawProducts, excludedBrands)
}

func (uc *UpsertProductsUseCase) execute(
	ctx context.Context,
	tenantID, sourceID uuid.UUID,
	jobID *uuid.UUID,
	rawProducts []scrapingport.RawProduct,
	excludedBrands []string,
) (created, updated int, err error) {
	// Filter out products whose brand (normalized: trim + lower) matches any excluded brand.
	if len(excludedBrands) > 0 {
		normalized := make([]string, len(excludedBrands))
		for i, b := range excludedBrands {
			normalized[i] = strings.ToLower(strings.TrimSpace(b))
		}
		filtered := rawProducts[:0]
		for _, raw := range rawProducts {
			brandNorm := strings.ToLower(strings.TrimSpace(raw.Brand))
			excluded := false
			for _, eb := range normalized {
				if brandNorm == eb {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, raw)
			}
		}
		filteredCount := len(rawProducts) - len(filtered)
		if filteredCount > 0 {
			uc.log(webdataport.WebdataEvent{
				Event:        "webdata.products_filtered",
				TenantID:     tenantID.String(),
				SourceID:     sourceID.String(),
				ProductCount: filteredCount,
			})
		}
		rawProducts = filtered
	}

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
					uc.log(webdataport.WebdataEvent{
						Event:    "webdata.product_save_failed",
						TenantID: tenantID.String(),
						SourceID: sourceID.String(),
						Reason:   saveErr.Error(),
					})
				}
			} else {
				existing.TouchLastSeen()
				if _, saveErr := uc.repo.Upsert(ctx, existing); saveErr == nil {
					updated++
				} else {
					uc.log(webdataport.WebdataEvent{
						Event:    "webdata.product_save_failed",
						TenantID: tenantID.String(),
						SourceID: sourceID.String(),
						Reason:   saveErr.Error(),
					})
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
			uc.log(webdataport.WebdataEvent{
				Event:    "webdata.product_save_failed",
				TenantID: tenantID.String(),
				SourceID: sourceID.String(),
				Reason:   fmt.Sprintf("construct: %v", newErr),
			})
			continue
		}

		if _, upsertErr := uc.repo.Upsert(ctx, product); upsertErr == nil {
			created++
			// Resolve business type per-product from its own category first;
			// fall back to the source-level autoAssignment when the per-product
			// resolver finds no match.
			var assignmentToApply *value_object.BusinessTypeAssignment
			if perProduct, ok := value_object.ResolveBusinessTypeFromProductCategory(raw.Category); ok {
				assignmentToApply = &perProduct
				uc.log(webdataport.WebdataEvent{
					Event:        "webdata.product_business_type_assigned",
					TenantID:     tenantID.String(),
					SourceID:     sourceID.String(),
					BusinessType: perProduct.BusinessTypeCode,
				})
			} else if autoAssignment != nil {
				assignmentToApply = autoAssignment
			}
			if assignmentToApply != nil {
				assignments := []value_object.BusinessTypeAssignment{*assignmentToApply}
				if btErr := uc.repo.SaveBusinessTypes(ctx, tenantID, product.ID, assignments); btErr != nil {
					uc.log(webdataport.WebdataEvent{
						Event:    "webdata.product_business_type_failed",
						TenantID: tenantID.String(),
						SourceID: sourceID.String(),
						Reason:   btErr.Error(),
					})
				}
				product.BusinessTypes = assignments
			}
			uc.trySyncToPIM(ctx, product)
		} else {
			uc.log(webdataport.WebdataEvent{
				Event:    "webdata.product_save_failed",
				TenantID: tenantID.String(),
				SourceID: sourceID.String(),
				Reason:   upsertErr.Error(),
			})
		}
	}

	return created, updated, nil
}

func (uc *UpsertProductsUseCase) trySyncToPIM(ctx context.Context, product *entity.ScrapedProduct) {
	if uc.pimSync == nil || !product.NeedsPIMSync() {
		return
	}
	if err := uc.pimSync.Execute(ctx, product, uc.sourceName); err != nil {
		uc.log(webdataport.WebdataEvent{
			Event:    "webdata.pim_sync_failed",
			TenantID: product.TenantID.String(),
			SourceID: product.SourceID.String(),
			Reason:   err.Error(),
		})
	}
}

