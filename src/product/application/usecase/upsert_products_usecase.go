package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type UpsertProductsUseCase struct {
	repo port.ProductRepository
}

func NewUpsertProductsUseCase(repo port.ProductRepository) *UpsertProductsUseCase {
	return &UpsertProductsUseCase{repo: repo}
}

// Execute upserts raw products from a scrape job. Returns count of saved (created+updated) products.
func (uc *UpsertProductsUseCase) Execute(ctx context.Context, tenantID, sourceID uuid.UUID, jobID *uuid.UUID, raw []scrapeport.RawProduct) (int, error) {
	saved := 0
	for _, r := range raw {
		if err := uc.upsertOne(ctx, tenantID, sourceID, jobID, r); err != nil {
			return saved, fmt.Errorf("upserting product %q: %w", r.Title, err)
		}
		saved++
	}
	return saved, nil
}

func (uc *UpsertProductsUseCase) upsertOne(ctx context.Context, tenantID, sourceID uuid.UUID, jobID *uuid.UUID, r scrapeport.RawProduct) error {
	hash := value_object.GenerateContentHash(tenantID, sourceID, r.Title, r.URL)

	existing, err := uc.repo.FindByContentHash(ctx, tenantID, sourceID, hash)
	if err == nil {
		// Product exists — check price change
		if r.Price != nil && existing.HasPriceChanged(*r.Price) {
			record := value_object.NewPriceRecord(tenantID, existing.ID, *r.Price)
			if err := uc.repo.SavePriceRecord(ctx, record); err != nil {
				return fmt.Errorf("saving price record: %w", err)
			}
			existing.UpdatePrice(*r.Price)
		} else {
			existing.TouchLastSeen()
		}
		if _, err := uc.repo.Upsert(ctx, existing); err != nil {
			return fmt.Errorf("updating existing product: %w", err)
		}
		return nil
	}

	// New product
	product, err := entity.NewScrapedProduct(entity.CreateProductParams{
		TenantID:        tenantID,
		SourceID:        sourceID,
		JobID:           jobID,
		Title:           r.Title,
		Price:           r.Price,
		OriginalPrice:   r.OriginalPrice,
		URL:             r.URL,
		ImageURL:        r.ImageURL,
		Description:     r.Description,
		Brand:           r.Brand,
		Category:        r.Category,
		SKU:             r.SKU,
		EAN:             r.EAN,
		InStock:         r.InStock,
		ContentHash:     hash,
	})
	if err != nil {
		return fmt.Errorf("building product: %w", err)
	}

	if _, err := uc.repo.Upsert(ctx, product); err != nil {
		return fmt.Errorf("inserting product: %w", err)
	}

	if r.Price != nil {
		record := value_object.NewPriceRecord(tenantID, product.ID, *r.Price)
		if err := uc.repo.SavePriceRecord(ctx, record); err != nil {
			return fmt.Errorf("saving initial price record: %w", err)
		}
	}

	return nil
}
