package application

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

type RunEnrichmentBatchRequest struct {
	BusinessType   string   `json:"business_type"`
	Limit          int      `json:"limit"`
	ForceReprocess bool     `json:"force_reprocess"`
	ProductIDs     []string `json:"product_ids,omitempty"`
}

type RunEnrichmentBatchResult struct {
	Processed int      `json:"processed"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	TotalCost int      `json:"total_cost_cents"`
	Errors    []string `json:"errors,omitempty"`
}

type RunEnrichmentBatchUseCase struct {
	pim      port.PIMClient
	sources  []port.ProductSource
	enhancer port.ImageEnhancer
	store    port.ImageStore
	repo     port.EnrichmentRepository
}

func NewRunEnrichmentBatchUseCase(
	pim port.PIMClient,
	sources []port.ProductSource,
	enhancer port.ImageEnhancer,
	store port.ImageStore,
	repo port.EnrichmentRepository,
) *RunEnrichmentBatchUseCase {
	return &RunEnrichmentBatchUseCase{
		pim:      pim,
		sources:  sources,
		enhancer: enhancer,
		store:    store,
		repo:     repo,
	}
}

const pageSize = 100

func (uc *RunEnrichmentBatchUseCase) Execute(ctx context.Context, req RunEnrichmentBatchRequest) (*RunEnrichmentBatchResult, error) {
	if len(req.ProductIDs) > 0 {
		return uc.executeOnDemand(ctx, req)
	}
	return uc.executeBatch(ctx, req)
}

func (uc *RunEnrichmentBatchUseCase) executeOnDemand(ctx context.Context, req RunEnrichmentBatchRequest) (*RunEnrichmentBatchResult, error) {
	if len(req.ProductIDs) > 100 {
		return nil, fmt.Errorf("on-demand enrichment acepta máximo 100 product_ids")
	}

	page, err := uc.pim.GetProductsByIDs(ctx, req.ProductIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching products by ids: %w", err)
	}

	result := &RunEnrichmentBatchResult{}
	for _, item := range page.Items {
		key := resolveKey(item)

		skipped, err := uc.shouldSkip(ctx, item.ID, req.ForceReprocess)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: skip check failed: %v", item.ID, err))
			continue
		}
		if skipped {
			result.Skipped++
			continue
		}

		cost, err := uc.processItem(ctx, item, key)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.ID, err))
			continue
		}

		result.Processed++
		result.TotalCost += cost
	}

	return result, nil
}

func (uc *RunEnrichmentBatchUseCase) executeBatch(ctx context.Context, req RunEnrichmentBatchRequest) (*RunEnrichmentBatchResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	result := &RunEnrichmentBatchResult{}
	offset := 0

	for result.Processed < limit {
		page, err := uc.pim.ListNeedingEnrichmentPage(ctx, req.BusinessType, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("listing enrichment queue at offset %d: %w", offset, err)
		}
		if len(page.Items) == 0 {
			break
		}

		for _, item := range page.Items {
			if result.Processed >= limit {
				break
			}
			key := resolveKey(item)

			skipped, err := uc.shouldSkip(ctx, item.ID, req.ForceReprocess)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: skip check failed: %v", item.ID, err))
				continue
			}
			if skipped {
				result.Skipped++
				continue
			}

			cost, err := uc.processItem(ctx, item, key)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.ID, err))
				continue
			}

			result.Processed++
			result.TotalCost += cost
		}

		offset += pageSize

		if len(page.Items) < pageSize {
			break
		}
	}

	return result, nil
}

func (uc *RunEnrichmentBatchUseCase) shouldSkip(ctx context.Context, key string, forceReprocess bool) (bool, error) {
	if forceReprocess {
		return false, nil
	}
	return uc.repo.IsResolved(ctx, key)
}

func (uc *RunEnrichmentBatchUseCase) processItem(ctx context.Context, item port.NeedsEnrichmentItem, key string) (int, error) {
	productData, err := uc.findProductData(ctx, item)
	if err != nil {
		return 0, fmt.Errorf("finding product data: %w", err)
	}
	if productData == nil || productData.ImageURL == "" {
		return 0, fmt.Errorf("no product data found in any source")
	}

	// Si el source devolvió un GTIN/EAN, usarlo como key (más preciso que el nombre)
	if productData.GTIN != "" {
		key = productData.GTIN
	} else if productData.EAN != "" {
		key = productData.EAN
	}

	imageBytes, mimeType, costCents, enhancer, err := uc.enhance(ctx, productData)
	if err != nil {
		return 0, fmt.Errorf("enhancing image: %w", err)
	}

	cdnURL, cdnKey, err := uc.store.Upload(ctx, key, imageBytes, mimeType)
	if err != nil {
		return 0, fmt.Errorf("uploading image: %w", err)
	}

	resolution := buildResolution(key, item, productData, cdnURL, cdnKey, enhancer, costCents)
	if err := uc.repo.Save(ctx, resolution); err != nil {
		return 0, fmt.Errorf("saving resolution: %w", err)
	}

	if err := uc.updatePIM(ctx, item, productData, cdnURL); err != nil {
		return 0, fmt.Errorf("updating pim: %w", err)
	}

	return costCents, nil
}

func (uc *RunEnrichmentBatchUseCase) findProductData(ctx context.Context, item port.NeedsEnrichmentItem) (*entity.ProductData, error) {
	rejectedURLs, err := uc.repo.GetRejectedURLs(ctx, item.ID)
	if err != nil {
		return nil, fmt.Errorf("getting rejected URLs: %w", err)
	}

	for _, src := range uc.sources {
		data, err := uc.fetchFromSource(ctx, src, item)
		if err != nil {
			// Source failed — try next source instead of aborting
			continue
		}
		if data == nil || data.ImageURL == "" {
			continue
		}
		if isRejectedURL(data.ImageURL, rejectedURLs) {
			continue
		}
		return data, nil
	}

	return nil, nil
}

func (uc *RunEnrichmentBatchUseCase) fetchFromSource(ctx context.Context, src port.ProductSource, item port.NeedsEnrichmentItem) (*entity.ProductData, error) {
	if item.EAN != "" {
		return src.FindByGTIN(ctx, item.EAN)
	}
	if item.Category != "" || item.BusinessType != "" {
		return src.FindByNameWithContext(ctx, item.Name, item.Category, item.BusinessType)
	}
	return src.FindByName(ctx, item.Name)
}

func isRejectedURL(imageURL string, rejectedURLs []string) bool {
	for _, rejected := range rejectedURLs {
		if rejected == imageURL {
			return true
		}
	}
	return false
}

func (uc *RunEnrichmentBatchUseCase) enhance(ctx context.Context, data *entity.ProductData) ([]byte, string, int, entity.Enhancer, error) {
	// El enhancer inyectado ya implementa la lógica de fallback (ChainEnhancer).
	// La clasificación de calidad la maneja internamente.
	imageBytes, mimeType, costCents, err := uc.enhancer.Enhance(ctx, data.ImageURL, data.Source)
	if err != nil {
		return nil, "", 0, entity.EnhancerPassthrough, err
	}

	enhancerType := entity.EnhancerPassthrough
	if data.Source == entity.SourceDALLE {
		enhancerType = entity.EnhancerDALLE
		costCents = 4
	} else if costCents > 0 {
		enhancerType = entity.EnhancerReplicate
	}
	return imageBytes, mimeType, costCents, enhancerType, nil
}

func (uc *RunEnrichmentBatchUseCase) updatePIM(ctx context.Context, item port.NeedsEnrichmentItem, data *entity.ProductData, cdnURL string) error {
	req := port.UpdateProductRequest{
		ImageURL: &cdnURL,
	}
	if data.GTIN != "" {
		req.GTIN = data.GTIN
	}
	if data.EAN != "" {
		req.EANCode = data.EAN
	}
	if data.Brand != "" {
		req.Brand = data.Brand
	}
	if data.Category != "" {
		req.Category = data.Category
	}
	return uc.pim.UpdateProduct(ctx, item.ID, req)
}

func resolveKey(item port.NeedsEnrichmentItem) string {
	if item.EAN != "" {
		return item.EAN
	}
	return sanitizeName(item.Name)
}

func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name) && i < 50; i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

func buildResolution(
	key string,
	item port.NeedsEnrichmentItem,
	data *entity.ProductData,
	cdnURL, cdnKey string,
	enhancer entity.Enhancer,
	costCents int,
) *entity.ImageResolution {
	now := time.Now()
	return &entity.ImageResolution{
		GTIN:         key,
		EAN:          data.EAN,
		ProductID:    item.ID,
		Source:       data.Source,
		RawURL:       data.ImageURL,
		CDNURL:       cdnURL,
		CDNKey:       cdnKey,
		Enhancer:     enhancer,
		CostCents:    costCents,
		QualityScore: 3,
		ResolvedAt:   now,
		UpdatedAt:    now,
	}
}
