package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type GetProductFiltersUseCase struct {
	repo port.ProductRepository
}

func NewGetProductFiltersUseCase(repo port.ProductRepository) *GetProductFiltersUseCase {
	return &GetProductFiltersUseCase{repo: repo}
}

func (uc *GetProductFiltersUseCase) Execute(ctx context.Context, tenantID uuid.UUID) (*port.ProductFilters, error) {
	return uc.repo.GetFilters(ctx, tenantID)
}
