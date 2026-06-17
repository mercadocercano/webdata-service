package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/database"
	webdataport "github.com/mercadocercano/webdata-service/src/webdata/domain/port"
)

// EnrichGlobalProductsUseCase consulta PIM por productos incompletos y crea scraping jobs
// para las fuentes de su business_type, enriqueciendo precio e imagen vía webdata.
type EnrichGlobalProductsUseCase struct {
	pimSyncer  port.PIMCatalogSyncer
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapingport.ScrapingJobRepository
	logger     webdataport.WebdataEventLogger
}

func NewEnrichGlobalProductsUseCase(
	pimSyncer port.PIMCatalogSyncer,
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapingport.ScrapingJobRepository,
) *EnrichGlobalProductsUseCase {
	return &EnrichGlobalProductsUseCase{
		pimSyncer:  pimSyncer,
		sourceRepo: sourceRepo,
		jobRepo:    jobRepo,
	}
}

// WithLogger inyecta el logger canónico (ADR-001). Nil-safe.
func (uc *EnrichGlobalProductsUseCase) WithLogger(logger webdataport.WebdataEventLogger) {
	uc.logger = logger
}

func (uc *EnrichGlobalProductsUseCase) logEvt(e webdataport.WebdataEvent) {
	if uc.logger != nil {
		uc.logger.Log(e)
	}
}

type EnrichGlobalProductsResult struct {
	ProductsNeedingEnrichment int      `json:"products_needing_enrichment"`
	JobsCreated               int      `json:"jobs_created"`
	BusinessTypesProcessed    []string `json:"business_types_processed"`
	Errors                    []string `json:"errors,omitempty"`
}

// Execute consulta PIM, agrupa por business_type y crea un scraping job por fuente activa.
func (uc *EnrichGlobalProductsUseCase) Execute(ctx context.Context, businessType string) (*EnrichGlobalProductsResult, error) {
	enrichItems, err := uc.pimSyncer.ListNeedingEnrichment(ctx, businessType, 500, 0)
	if err != nil {
		return nil, fmt.Errorf("fetching products needing enrichment from PIM: %w", err)
	}

	btSet := uc.collectBusinessTypes(enrichItems.Items)

	result := &EnrichGlobalProductsResult{
		ProductsNeedingEnrichment: enrichItems.Total,
	}

	for bt := range btSet {
		result.BusinessTypesProcessed = append(result.BusinessTypesProcessed, bt)
		created, errs := uc.triggerSourcesForBusinessType(ctx, bt)
		result.JobsCreated += created
		result.Errors = append(result.Errors, errs...)
	}

	return result, nil
}

func (uc *EnrichGlobalProductsUseCase) collectBusinessTypes(items []port.NeedsEnrichmentItem) map[string]bool {
	btSet := make(map[string]bool)
	for _, item := range items {
		if item.BusinessType != "" {
			btSet[item.BusinessType] = true
		}
	}
	return btSet
}

func (uc *EnrichGlobalProductsUseCase) triggerSourcesForBusinessType(ctx context.Context, businessType string) (created int, errs []string) {
	isActive := true
	sources, _, err := uc.sourceRepo.FindAll(ctx, database.GlobalTenantID, sourceport.SourceFilter{
		Category: businessType,
		IsActive: &isActive,
	})
	if err != nil {
		uc.logEvt(webdataport.WebdataEvent{
			Event:        "webdata.enrichment_sources_failed",
			BusinessType: businessType,
			Reason:       err.Error(),
		})
		errs = append(errs, fmt.Sprintf("sources for %s: %v", businessType, err))
		return
	}

	for _, source := range sources {
		job, err := entity.NewScrapingJob(uuid.Nil, source.ID, "enrichment")
		if err != nil {
			uc.logEvt(webdataport.WebdataEvent{
				Event:        "webdata.enrichment_job_failed",
				SourceID:     source.ID.String(),
				BusinessType: businessType,
				Reason:       fmt.Sprintf("create job: %v", err),
			})
			continue
		}
		if err := uc.jobRepo.Save(ctx, job); err != nil {
			uc.logEvt(webdataport.WebdataEvent{
				Event:        "webdata.enrichment_job_failed",
				SourceID:     source.ID.String(),
				BusinessType: businessType,
				Reason:       fmt.Sprintf("save job: %v", err),
			})
			continue
		}
		created++
	}
	if created > 0 {
		uc.logEvt(webdataport.WebdataEvent{
			Event:        "webdata.enrichment_completed",
			BusinessType: businessType,
			JobsCreated:  created,
		})
	}
	return
}
