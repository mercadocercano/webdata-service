package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type BulkDeleteProductsUseCase struct {
	repo port.ProductRepository
}

func NewBulkDeleteProductsUseCase(repo port.ProductRepository) *BulkDeleteProductsUseCase {
	return &BulkDeleteProductsUseCase{repo: repo}
}

func (uc *BulkDeleteProductsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int64, error) {
	return uc.repo.BulkSoftDelete(ctx, tenantID, ids)
}
