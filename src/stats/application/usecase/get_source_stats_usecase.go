package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/stats/application/response"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type GetSourceStatsUseCase struct {
	sourceRepo  sourceport.SourceRepository
	jobRepo     scrapeport.ScrapingJobRepository
	productRepo productport.ProductRepository
}

func NewGetSourceStatsUseCase(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapeport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *GetSourceStatsUseCase {
	return &GetSourceStatsUseCase{sourceRepo: sourceRepo, jobRepo: jobRepo, productRepo: productRepo}
}

func (uc *GetSourceStatsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) ([]response.SourceStatsResponse, error) {
	sources, _, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{PageSize: 10000})
	if err != nil {
		return nil, fmt.Errorf("fetching sources: %w", err)
	}

	result := make([]response.SourceStatsResponse, 0, len(sources))
	for _, s := range sources {
		sourceID := s.ID
		_, totalJobs, _ := uc.jobRepo.FindAll(ctx, tenantID, scrapeport.JobFilter{SourceID: &sourceID, PageSize: 1})
		_, totalProducts, _ := uc.productRepo.FindAll(ctx, tenantID, productport.ProductFilter{SourceID: &sourceID, PageSize: 1})

		result = append(result, response.SourceStatsResponse{
			SourceID:      s.ID,
			Name:          s.Name,
			TotalProducts: totalProducts,
			TotalJobs:     totalJobs,
			HealthScore:   s.HealthScore,
		})
	}

	return result, nil
}
