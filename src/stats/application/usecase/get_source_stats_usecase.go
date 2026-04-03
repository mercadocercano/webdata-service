package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/stats/application/response"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type GetSourceStatsUseCase struct {
	sourceRepo sourceport.SourceRepository
}

func NewGetSourceStatsUseCase(sourceRepo sourceport.SourceRepository) *GetSourceStatsUseCase {
	return &GetSourceStatsUseCase{sourceRepo: sourceRepo}
}

func (uc *GetSourceStatsUseCase) Execute(ctx context.Context, tenantID uuid.UUID) ([]response.SourceStatsResponse, error) {
	sources, _, err := uc.sourceRepo.FindAll(ctx, tenantID, sourceport.SourceFilter{PageSize: 9999})
	if err != nil {
		return nil, err
	}

	result := make([]response.SourceStatsResponse, len(sources))
	for i, s := range sources {
		result[i] = response.SourceStatsResponse{
			SourceID:       s.ID.String(),
			Name:           s.Name,
			HealthScore:    s.HealthScore,
			TotalRuns:      s.TotalRuns,
			SuccessfulRuns: s.SuccessfulRuns,
			FailedRuns:     s.FailedRuns,
			IsActive:       s.IsActive,
		}
	}
	return result, nil
}
