package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	vo "github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type BulkAssignBusinessTypeResult struct {
	Assigned int
	Failed   int
	Errors   []string
}

type BulkAssignBusinessTypeUseCase struct {
	repo port.ProductRepository
}

func NewBulkAssignBusinessTypeUseCase(repo port.ProductRepository) *BulkAssignBusinessTypeUseCase {
	return &BulkAssignBusinessTypeUseCase{repo: repo}
}

func (uc *BulkAssignBusinessTypeUseCase) Execute(
	ctx context.Context,
	tenantID uuid.UUID,
	productIDs []uuid.UUID,
	assignment vo.BusinessTypeAssignment,
) (BulkAssignBusinessTypeResult, error) {
	result := BulkAssignBusinessTypeResult{}

	for _, pid := range productIDs {
		existing, err := uc.repo.FindBusinessTypesForProduct(ctx, tenantID, pid)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", pid, err))
			continue
		}

		// Check if already assigned
		alreadyAssigned := false
		for _, e := range existing {
			if e.BusinessTypeCode == assignment.BusinessTypeCode {
				alreadyAssigned = true
				break
			}
		}

		if alreadyAssigned {
			result.Assigned++ // count as success (idempotent)
			continue
		}

		updated := append(existing, assignment)
		if err := uc.repo.SaveBusinessTypes(ctx, tenantID, pid, updated); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", pid, err))
			continue
		}
		result.Assigned++
	}

	return result, nil
}
