package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type RetryJobUseCase struct {
	repo scrapingport.ScrapingJobRepository
}

func NewRetryJobUseCase(repo scrapingport.ScrapingJobRepository) *RetryJobUseCase {
	return &RetryJobUseCase{repo: repo}
}

func (uc *RetryJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.JobResponse, error) {
	original, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.JobResponse{}, err
	}

	if !original.CanRetry() {
		return response.JobResponse{}, fmt.Errorf("job has reached max retries (%d)", original.MaxRetries)
	}

	newJob, err := entity.NewScrapingJob(tenantID, original.SourceID, original.TriggerType.Value())
	if err != nil {
		return response.JobResponse{}, err
	}
	newJob.RetryCount = original.RetryCount + 1
	newJob.MaxRetries = original.MaxRetries

	if err := uc.repo.Save(ctx, newJob); err != nil {
		return response.JobResponse{}, err
	}

	return response.FromJob(newJob), nil
}
