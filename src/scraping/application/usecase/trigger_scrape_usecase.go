package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type TriggerScrapeUseCase struct {
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapeport.ScrapingJobRepository
}

func NewTriggerScrapeUseCase(sourceRepo sourceport.SourceRepository, jobRepo scrapeport.ScrapingJobRepository) *TriggerScrapeUseCase {
	return &TriggerScrapeUseCase{sourceRepo: sourceRepo, jobRepo: jobRepo}
}

func (uc *TriggerScrapeUseCase) Execute(ctx context.Context, tenantID, sourceID uuid.UUID) (response.JobResponse, error) {
	if _, err := uc.sourceRepo.FindByID(ctx, tenantID, sourceID); err != nil {
		return response.JobResponse{}, fmt.Errorf("source not found: %w", err)
	}

	job, err := entity.NewScrapingJob(tenantID, sourceID, "manual")
	if err != nil {
		return response.JobResponse{}, fmt.Errorf("creating job: %w", err)
	}

	if err := uc.jobRepo.Save(ctx, job); err != nil {
		return response.JobResponse{}, fmt.Errorf("saving job: %w", err)
	}

	return response.FromJob(job), nil
}
