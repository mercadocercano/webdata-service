package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type ListSourcesUseCase struct {
	repo port.SourceRepository
}

func NewListSourcesUseCase(repo port.SourceRepository) *ListSourcesUseCase {
	return &ListSourcesUseCase{repo: repo}
}

type ListSourcesResult struct {
	Sources  []response.SourceResponse
	Total    int
	Page     int
	PageSize int
}

func (uc *ListSourcesUseCase) Execute(ctx context.Context, tenantID uuid.UUID, filter port.SourceFilter) (ListSourcesResult, error) {
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	sources, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListSourcesResult{}, fmt.Errorf("listing sources: %w", err)
	}

	result := make([]response.SourceResponse, len(sources))
	for i, s := range sources {
		result[i] = response.FromSource(s)
	}

	return ListSourcesResult{
		Sources:  result,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}
