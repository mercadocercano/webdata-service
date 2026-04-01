package usecase

import (
	"context"

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
	return uc.repo.Delete(ctx, tenantID, id)
}
