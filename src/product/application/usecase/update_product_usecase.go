package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type UpdateProductUseCase struct {
	repo port.ProductRepository
}

func NewUpdateProductUseCase(repo port.ProductRepository) *UpdateProductUseCase {
	return &UpdateProductUseCase{repo: repo}
}

func (uc *UpdateProductUseCase) Execute(ctx context.Context, tenantID, productID uuid.UUID, isBlocked bool) error {
	return uc.repo.UpdateBlocked(ctx, tenantID, productID, isBlocked)
}
