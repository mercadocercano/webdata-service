package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type DeleteSourceUseCase struct {
	repo port.SourceRepository
}

func NewDeleteSourceUseCase(repo port.SourceRepository) *DeleteSourceUseCase {
	return &DeleteSourceUseCase{repo: repo}
}

func (uc *DeleteSourceUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
	if _, err := uc.repo.FindByID(ctx, tenantID, id); err != nil {
		return fmt.Errorf("source not found: %w", err)
	}
	if err := uc.repo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("deleting source: %w", err)
	}
	return nil
}
