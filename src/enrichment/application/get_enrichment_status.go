package application

import (
	"context"
	"fmt"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

type GetEnrichmentStatusUseCase struct {
	repo port.EnrichmentRepository
}

func NewGetEnrichmentStatusUseCase(repo port.EnrichmentRepository) *GetEnrichmentStatusUseCase {
	return &GetEnrichmentStatusUseCase{repo: repo}
}

func (uc *GetEnrichmentStatusUseCase) Execute(ctx context.Context) (*port.EnrichmentStatus, error) {
	status, err := uc.repo.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting enrichment status: %w", err)
	}
	return status, nil
}
