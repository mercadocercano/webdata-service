package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type GetSourceUseCase struct {
	repo port.SourceRepository
}

func NewGetSourceUseCase(repo port.SourceRepository) *GetSourceUseCase {
	return &GetSourceUseCase{repo: repo}
}

func (uc *GetSourceUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID) (response.SourceResponse, error) {
	source, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.SourceResponse{}, err
	}
	return response.FromSource(source), nil
}
