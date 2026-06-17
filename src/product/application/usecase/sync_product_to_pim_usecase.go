package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	webdataport "github.com/mercadocercano/webdata-service/src/webdata/domain/port"
)

type SyncProductToPIMUseCase struct {
	repo   port.ProductRepository
	syncer port.PIMCatalogSyncer
	logger webdataport.WebdataEventLogger
}

func NewSyncProductToPIMUseCase(repo port.ProductRepository, syncer port.PIMCatalogSyncer) *SyncProductToPIMUseCase {
	return &SyncProductToPIMUseCase{repo: repo, syncer: syncer}
}

// WithLogger inyecta el logger canónico (ADR-001). Nil-safe.
func (uc *SyncProductToPIMUseCase) WithLogger(logger webdataport.WebdataEventLogger) {
	uc.logger = logger
}

func (uc *SyncProductToPIMUseCase) logEvt(e webdataport.WebdataEvent) {
	if uc.logger != nil {
		uc.logger.Log(e)
	}
}

func (uc *SyncProductToPIMUseCase) Execute(ctx context.Context, product *entity.ScrapedProduct, sourceName string) error {
	if !product.NeedsPIMSync() {
		return nil
	}

	existing, err := uc.findExisting(ctx, product)
	if err != nil {
		return fmt.Errorf("searching PIM: %w", err)
	}

	if existing != nil {
		return uc.updateExisting(ctx, product, existing)
	}

	return uc.createNew(ctx, product, sourceName)
}

func (uc *SyncProductToPIMUseCase) findExisting(ctx context.Context, product *entity.ScrapedProduct) (*port.PIMGlobalProduct, error) {
	if value_object.IsValidEAN13(product.EAN) {
		found, err := uc.syncer.SearchByEAN(ctx, product.TenantID, product.EAN)
		if err != nil {
			return nil, err
		}
		if found != nil {
			return found, nil
		}
	}
	return uc.syncer.SearchByNameBrand(ctx, product.TenantID, product.Title, product.Brand)
}

func (uc *SyncProductToPIMUseCase) updateExisting(ctx context.Context, product *entity.ScrapedProduct, existing *port.PIMGlobalProduct) error {
	req := port.UpdatePIMProductRequest{
		Price:    product.Price,
		ImageURL: product.ImageURL,
	}
	if err := uc.syncer.Update(ctx, product.TenantID, existing.ID, req); err != nil {
		return fmt.Errorf("updating PIM product: %w", err)
	}
	return uc.markSynced(ctx, product)
}

func (uc *SyncProductToPIMUseCase) createNew(ctx context.Context, product *entity.ScrapedProduct, sourceName string) error {
	source := "scraper"
	if sourceName != "" {
		source = "scraper_" + sourceName
	}

	bt := primaryBusinessType(product)

	req := port.CreatePIMProductRequest{
		EAN:          value_object.NormalizeEANForSync(product.EAN),
		Name:         product.Title,
		Brand:        product.Brand,
		Category:     product.Category,
		Price:        product.Price,
		ImageURL:     product.ImageURL,
		Source:       source,
		SourceURL:    product.URL,
		BusinessType: bt,
	}

	if _, err := uc.syncer.Create(ctx, product.TenantID, req); err != nil {
		return fmt.Errorf("creating PIM product: %w", err)
	}
	return uc.markSynced(ctx, product)
}

func (uc *SyncProductToPIMUseCase) markSynced(ctx context.Context, product *entity.ScrapedProduct) error {
	product.MarkSyncedToPIM()
	if err := uc.repo.MarkSyncedToPIM(ctx, product.TenantID, product.ID); err != nil {
		uc.logEvt(webdataport.WebdataEvent{
			Event:    "webdata.pim_sync_mark_failed",
			TenantID: product.TenantID.String(),
			SourceID: product.SourceID.String(),
			Reason:   err.Error(),
		})
		return nil
	}
	return nil
}

func primaryBusinessType(product *entity.ScrapedProduct) string {
	if len(product.BusinessTypes) > 0 {
		return product.BusinessTypes[0].BusinessTypeCode
	}
	return ""
}

// SyncBatch syncs multiple products. Errors are logged per product, not propagated.
func (uc *SyncProductToPIMUseCase) SyncBatch(ctx context.Context, products []*entity.ScrapedProduct, sourceName string) (synced, skipped, failed int) {
	for _, p := range products {
		if !p.NeedsPIMSync() {
			skipped++
			continue
		}
		if err := uc.Execute(ctx, p, sourceName); err != nil {
			uc.logEvt(webdataport.WebdataEvent{
				Event:    "webdata.pim_sync_failed",
				TenantID: p.TenantID.String(),
				SourceID: p.SourceID.String(),
				Reason:   err.Error(),
			})
			failed++
			continue
		}
		synced++
	}
	return
}

// SyncPending finds and syncs all products pending PIM sync for a tenant.
// It paginates through all pending products (NotSyncedToPIM=true) in batches
// of syncPendingPageSize until no more products are found.
const syncPendingPageSize = 200

func (uc *SyncProductToPIMUseCase) SyncPending(ctx context.Context, tenantID uuid.UUID, sourceName string) (synced, failed int, err error) {
	// Always fetch page=1 with NotSyncedToPIM=true: as products get marked synced,
	// the "pending" set shrinks and page=1 always returns the next unprocessed batch.
	// Incrementing the page number here would skip products (N products just synced
	// are removed from the filter, shifting what page=2 sees).
	const maxIterations = 500 // safety net: prevents infinite loops when every product in a batch fails
	for iter := 0; iter < maxIterations; iter++ {
		filter := port.ProductFilter{
			NotSyncedToPIM: true,
			Page:           1,
			PageSize:       syncPendingPageSize,
		}
		products, _, fetchErr := uc.repo.FindAll(ctx, tenantID, filter)
		if fetchErr != nil {
			return synced, failed, fmt.Errorf("fetching pending products: %w", fetchErr)
		}
		if len(products) == 0 {
			break
		}

		batchSynced := 0
		for _, p := range products {
			if !p.NeedsPIMSync() {
				batchSynced++ // counts as "consumed" from the pending set (it's not actually pending)
				continue
			}
			if syncErr := uc.Execute(ctx, p, sourceName); syncErr != nil {
				uc.logEvt(webdataport.WebdataEvent{
					Event:    "webdata.pim_sync_failed",
					TenantID: p.TenantID.String(),
					SourceID: p.SourceID.String(),
					Reason:   syncErr.Error(),
				})
				failed++
				continue
			}
			synced++
			batchSynced++
		}

		// If no product in this batch made progress (all failed), stop to avoid a loop.
		if batchSynced == 0 {
			uc.logEvt(webdataport.WebdataEvent{
				Event:     "webdata.sync_batch_stalled",
				BatchSize: len(products),
			})
			break
		}
	}

	// After batch sync, trigger PIM to refresh template products from global_products
	if synced > 0 {
		if err := uc.syncer.RefreshTemplateProducts(ctx); err != nil {
			uc.logEvt(webdataport.WebdataEvent{
				Event:  "webdata.pim_refresh_failed",
				Reason: err.Error(),
				Synced: synced,
			})
		} else {
			uc.logEvt(webdataport.WebdataEvent{
				Event:  "webdata.pim_sync_completed",
				Synced: synced,
			})
		}
	}

	return synced, failed, nil
}
