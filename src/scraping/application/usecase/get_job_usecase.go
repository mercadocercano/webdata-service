package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type GetJobUseCase struct {
	repo scrapingport.ScrapingJobRepository
}

func NewGetJobUseCase(repo scrapingport.ScrapingJobRepository) *GetJobUseCase {
	return &GetJobUseCase{repo: repo}
}

func (uc *GetJobUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.JobResponse, error) {
	job, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.JobResponse{}, err
	}
	return response.FromJob(job), nil
}
