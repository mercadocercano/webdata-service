package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type TriggerScrapeUseCase struct {
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapingport.ScrapingJobRepository
}

func NewTriggerScrapeUseCase(sourceRepo sourceport.SourceRepository, jobRepo scrapingport.ScrapingJobRepository) *TriggerScrapeUseCase {
	return &TriggerScrapeUseCase{sourceRepo: sourceRepo, jobRepo: jobRepo}
}

func (uc *TriggerScrapeUseCase) Execute(ctx context.Context, tenantID, sourceID uuid.UUID) (response.JobResponse, error) {
	_, err := uc.sourceRepo.FindByID(ctx, tenantID, sourceID)
	if err != nil {
		return response.JobResponse{}, err
	}

	job, err := entity.NewScrapingJob(tenantID, sourceID, "manual")
	if err != nil {
		return response.JobResponse{}, err
	}

	if err := uc.jobRepo.Save(ctx, job); err != nil {
		return response.JobResponse{}, err
	}

	return response.FromJob(job), nil
}
