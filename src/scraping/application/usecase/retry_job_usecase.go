package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type RetryJobUseCase struct {
	jobRepo port.ScrapingJobRepository
}

func NewRetryJobUseCase(jobRepo port.ScrapingJobRepository) *RetryJobUseCase {
	return &RetryJobUseCase{jobRepo: jobRepo}
}

func (uc *RetryJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.JobResponse, error) {
	original, err := uc.jobRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.JobResponse{}, fmt.Errorf("job not found: %w", err)
	}
	if !original.Status.IsTerminal() {
		return response.JobResponse{}, fmt.Errorf("can only retry terminal jobs, current status: %s", original.Status.Value())
	}
	if !original.CanRetry() {
		return response.JobResponse{}, fmt.Errorf("job has reached max retries (%d)", original.MaxRetries)
	}

	newJob, err := entity.NewScrapingJob(tenantID, original.SourceID, original.TriggerType.Value())
	if err != nil {
		return response.JobResponse{}, fmt.Errorf("creating retry job: %w", err)
	}
	newJob.RetryCount = original.RetryCount + 1

	if err := uc.jobRepo.Save(ctx, newJob); err != nil {
		return response.JobResponse{}, fmt.Errorf("saving retry job: %w", err)
	}

	return response.FromJob(newJob), nil
}
