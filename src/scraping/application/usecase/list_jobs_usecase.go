package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type ListJobsUseCase struct {
	repo port.ScrapingJobRepository
}

func NewListJobsUseCase(repo port.ScrapingJobRepository) *ListJobsUseCase {
	return &ListJobsUseCase{repo: repo}
}

type ListJobsResult struct {
	Jobs     []response.JobResponse
	Total    int
	Page     int
	PageSize int
}

func (uc *ListJobsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, filter port.JobFilter) (ListJobsResult, error) {
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	jobs, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListJobsResult{}, fmt.Errorf("listing jobs: %w", err)
	}

	result := make([]response.JobResponse, len(jobs))
	for i, j := range jobs {
		result[i] = response.FromJob(j)
	}

	return ListJobsResult{Jobs: result, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}
