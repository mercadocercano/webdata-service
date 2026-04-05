package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type RemoveBusinessTypeUseCase struct {
	repo port.ProductRepository
}

func NewRemoveBusinessTypeUseCase(repo port.ProductRepository) *RemoveBusinessTypeUseCase {
	return &RemoveBusinessTypeUseCase{repo: repo}
}

func (uc *RemoveBusinessTypeUseCase) Execute(ctx context.Context, tenantID, productID uuid.UUID, code string) error {
	_, err := uc.repo.FindByID(ctx, tenantID, productID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	return uc.repo.RemoveBusinessType(ctx, tenantID, productID, code)
}
