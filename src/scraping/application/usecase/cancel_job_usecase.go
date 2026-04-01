package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type CancelJobUseCase struct {
	repo port.ScrapingJobRepository
}

func NewCancelJobUseCase(repo port.ScrapingJobRepository) *CancelJobUseCase {
	return &CancelJobUseCase{repo: repo}
}

func (uc *CancelJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
	job, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}
	if err := job.Cancel(); err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}
	if err := uc.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("saving cancelled job: %w", err)
	}
	return nil
}
