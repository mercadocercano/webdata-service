package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	vo "github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type AssignBusinessTypesUseCase struct {
	repo port.ProductRepository
}

func NewAssignBusinessTypesUseCase(repo port.ProductRepository) *AssignBusinessTypesUseCase {
	return &AssignBusinessTypesUseCase{repo: repo}
}

func (uc *AssignBusinessTypesUseCase) Execute(ctx context.Context, tenantID, productID uuid.UUID, assignments []vo.BusinessTypeAssignment) error {
	_, err := uc.repo.FindByID(ctx, tenantID, productID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	return uc.repo.SaveBusinessTypes(ctx, tenantID, productID, assignments)
}
