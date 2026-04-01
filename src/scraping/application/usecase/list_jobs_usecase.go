package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/response"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type ListJobsResult struct {
	Items      []response.JobResponse
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

type ListJobsUseCase struct {
	repo scrapingport.ScrapingJobRepository
}

func NewListJobsUseCase(repo scrapingport.ScrapingJobRepository) *ListJobsUseCase {
	return &ListJobsUseCase{repo: repo}
}

func (uc *ListJobsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, filter scrapingport.JobFilter) (ListJobsResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	jobs, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListJobsResult{}, err
	}

	items := make([]response.JobResponse, len(jobs))
	for i, j := range jobs {
		items[i] = response.FromJob(j)
	}

	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	return ListJobsResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}
