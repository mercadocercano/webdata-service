package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/stats/application/response"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type GetStatsUseCase struct {
	sourceRepo  sourceport.SourceRepository
	jobRepo     scrapeport.ScrapingJobRepository
	productRepo productport.ProductRepository
}

func NewGetStatsUseCase(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapeport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *GetStatsUseCase {
	return &GetStatsUseCase{sourceRepo: sourceRepo, jobRepo: jobRepo, productRepo: productRepo}
}

func (uc *GetStatsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) (response.StatsResponse, error) {
	sources, totalSources, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{PageSize: 10000})
	if err != nil {
		return response.StatsResponse{}, fmt.Errorf("fetching sources: %w", err)
	}

	activeSources := 0
	for _, s := range sources {
		if s.IsActive {
			activeSources++
		}
	}

	_, totalJobs, err := uc.jobRepo.FindAll(ctx, tenantID, scrapeport.JobFilter{PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, fmt.Errorf("fetching jobs: %w", err)
	}

	_, totalProducts, err := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{PageSize: 1})
	if err != nil {
		return response.StatsResponse{}, fmt.Errorf("fetching products: %w", err)
	}

	successRate := 0.0
	if totalSources > 0 {
		totalScore := 0.0
		for _, s := range sources {
			totalScore += s.HealthScore
		}
		successRate = totalScore / float64(totalSources)
	}

	return response.StatsResponse{
		TotalSources:  totalSources,
		ActiveSources: activeSources,
		TotalProducts: totalProducts,
		TotalJobs:     totalJobs,
		SuccessRate:   successRate,
	}, nil
}
