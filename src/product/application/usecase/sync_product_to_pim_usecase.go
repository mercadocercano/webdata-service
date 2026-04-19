package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type SyncProductToPIMUseCase struct {
	repo   port.ProductRepository
	syncer port.PIMCatalogSyncer
}

func NewSyncProductToPIMUseCase(repo port.ProductRepository, syncer port.PIMCatalogSyncer) *SyncProductToPIMUseCase {
	return &SyncProductToPIMUseCase{repo: repo, syncer: syncer}
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
	if product.EAN != "" {
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
		EAN:          product.EAN,
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
		log.Printf("[sync-pim] warning: synced but failed to mark: product=%s err=%v", product.ID, err)
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
			log.Printf("[sync-pim] error syncing product %s: %v", p.ID, err)
			failed++
			continue
		}
		synced++
	}
	return
}

// SyncPending finds and syncs all products pending PIM sync for a tenant.
func (uc *SyncProductToPIMUseCase) SyncPending(ctx context.Context, tenantID uuid.UUID, sourceName string) (synced, failed int, err error) {
	filter := port.ProductFilter{Page: 1, PageSize: 500}
	products, _, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return 0, 0, fmt.Errorf("fetching products: %w", err)
	}

	for _, p := range products {
		if !p.NeedsPIMSync() {
			continue
		}
		if syncErr := uc.Execute(ctx, p, sourceName); syncErr != nil {
			log.Printf("[sync-pim] error syncing product %s: %v", p.ID, syncErr)
			failed++
			continue
		}
		synced++
	}

	// After batch sync, trigger PIM to refresh template products from global_products
	if synced > 0 {
		if err := uc.syncer.RefreshTemplateProducts(ctx); err != nil {
			log.Printf("[sync-pim] warning: refresh template products failed: %v", err)
		} else {
			log.Printf("[sync-pim] template products refreshed after syncing %d products", synced)
		}
	}

	return synced, failed, nil
}
