package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type GetJobUseCase struct {
	repo port.ScrapingJobRepository
}

func NewGetJobUseCase(repo port.ScrapingJobRepository) *GetJobUseCase {
	return &GetJobUseCase{repo: repo}
}

func (uc *GetJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.JobResponse, error) {
	job, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.JobResponse{}, fmt.Errorf("job not found: %w", err)
	}
	return response.FromJob(job), nil
}
