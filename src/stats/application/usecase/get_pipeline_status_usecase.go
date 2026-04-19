package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/stats/application/response"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type GetPipelineStatusUseCase struct {
	sourceRepo  sourceport.SourceRepository
	jobRepo     scrapingport.ScrapingJobRepository
	productRepo productport.ProductRepository
}

func NewGetPipelineStatusUseCase(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapingport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *GetPipelineStatusUseCase {
	return &GetPipelineStatusUseCase{
		sourceRepo:  sourceRepo,
		jobRepo:     jobRepo,
		productRepo: productRepo,
	}
}

func (uc *GetPipelineStatusUseCase) Execute(ctx context.Context, tenantID uuid.UUID, category string) (response.PipelineStatusResponse, error) {
	isActive := true

	// Sources by category
	allSources, _, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{Category: category, PageSize: 9999})
	if err != nil {
		return response.PipelineStatusResponse{}, err
	}
	activeSources, _, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{Category: category, IsActive: &isActive, PageSize: 9999})
	if err != nil {
		return response.PipelineStatusResponse{}, err
	}

	healthy, failing := 0, 0
	var lastScrape, nextScrape *time.Time
	for _, s := range activeSources {
		if s.HealthScore >= 0.5 {
			healthy++
		} else {
			failing++
		}
		if s.LastRunAt != nil && (lastScrape == nil || s.LastRunAt.After(*lastScrape)) {
			t := *s.LastRunAt
			lastScrape = &t
		}
		if s.NextRunAt != nil && (nextScrape == nil || s.NextRunAt.Before(*nextScrape)) {
			t := *s.NextRunAt
			nextScrape = &t
		}
	}

	// Products — total
	_, totalScraped, err := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{PageSize: 1})
	if err != nil {
		return response.PipelineStatusResponse{}, err
	}

	// Products without image
	_, withoutImage, err := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{WithoutImage: true, PageSize: 1})
	if err != nil {
		return response.PipelineStatusResponse{}, err
	}

	// Products not synced to PIM
	_, pendingSync, err := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{NotSyncedToPIM: true, PageSize: 1})
	if err != nil {
		return response.PipelineStatusResponse{}, err
	}

	withImage := totalScraped - withoutImage
	synced := totalScraped - pendingSync

	return response.PipelineStatusResponse{
		BusinessType: category,
		Sources: response.SourcesSummary{
			Total:   len(allSources),
			Active:  len(activeSources),
			Healthy: healthy,
			Failing: failing,
		},
		Products: response.ProductsSummary{
			Scraped:      totalScraped,
			WithImage:    withImage,
			WithoutImage: withoutImage,
			SyncedToPIM:  synced,
			PendingSync:  pendingSync,
		},
		LastScrape: lastScrape,
		NextScrape: nextScrape,
	}, nil
}
