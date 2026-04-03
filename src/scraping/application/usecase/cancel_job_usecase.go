package usecase

import (
	"context"

	"github.com/google/uuid"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type CancelJobUseCase struct {
	repo scrapingport.ScrapingJobRepository
}

func NewCancelJobUseCase(repo scrapingport.ScrapingJobRepository) *CancelJobUseCase {
	return &CancelJobUseCase{repo: repo}
}

func (uc *CancelJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
	job, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := job.Cancel(); err != nil {
		return err
	}
	return uc.repo.Update(ctx, job)
}
