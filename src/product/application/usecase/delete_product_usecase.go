package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type DeleteProductUseCase struct {
	repo port.ProductRepository
}

func NewDeleteProductUseCase(repo port.ProductRepository) *DeleteProductUseCase {
	return &DeleteProductUseCase{repo: repo}
}

func (uc *DeleteProductUseCase) Execute(ctx context.Context, tenantID, productID uuid.UUID) error {
	return uc.repo.SoftDelete(ctx, tenantID, productID)
}
