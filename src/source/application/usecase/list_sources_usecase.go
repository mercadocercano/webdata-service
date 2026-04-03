package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type ListSourcesResult struct {
	Items      []response.SourceResponse
	Sources    []response.SourceResponse // alias, kept for test compatibility
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

type ListSourcesUseCase struct {
	repo port.SourceRepository
}

func NewListSourcesUseCase(repo port.SourceRepository) *ListSourcesUseCase {
	return &ListSourcesUseCase{repo: repo}
}

func (uc *ListSourcesUseCase) Execute(ctx context.Context, tenantID uuid.UUID, filter port.SourceFilter) (ListSourcesResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	sources, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListSourcesResult{}, err
	}

	items := make([]response.SourceResponse, len(sources))
	for i, s := range sources {
		items[i] = response.FromSource(s)
	}

	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	return ListSourcesResult{
		Items:      items,
		Sources:    items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}
