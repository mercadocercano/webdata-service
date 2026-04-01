package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/response"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type GetProductUseCase struct {
	repo port.ProductRepository
}

func NewGetProductUseCase(repo port.ProductRepository) *GetProductUseCase {
	return &GetProductUseCase{repo: repo}
}

func (uc *GetProductUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.ProductResponse, error) {
	product, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.ProductResponse{}, err
	}
	return response.FromProduct(product), nil
}
