package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/request"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
)

type UpdateSourceUseCase struct {
	repo port.SourceRepository
}

func NewUpdateSourceUseCase(repo port.SourceRepository) *UpdateSourceUseCase {
	return &UpdateSourceUseCase{repo: repo}
}

func (uc *UpdateSourceUseCase) Execute(ctx context.Context, tenantID, id uuid.UUID, req request.UpdateSourceRequest) (response.SourceResponse, error) {
	source, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return response.SourceResponse{}, fmt.Errorf("source not found: %w", err)
	}

	if req.Name != nil {
		source.Name = *req.Name
	}
	if req.BaseURL != nil {
		source.BaseURL = *req.BaseURL
	}
	if req.Category != nil {
		source.Category = *req.Category
	}
	if req.City != nil {
		source.City = *req.City
	}
	if req.Priority != nil {
		p, err := value_object.NewSourcePriority(*req.Priority)
		if err != nil {
			return response.SourceResponse{}, err
		}
		source.Priority = p
	}
	if req.FirecrawlMethod != nil {
		source.FirecrawlMethod = *req.FirecrawlMethod
	}
	if req.CronExpression != nil {
		source.CronExpression = *req.CronExpression
	}
	if req.Notes != nil {
		source.Notes = *req.Notes
	}
	if req.IsActive != nil {
		source.IsActive = *req.IsActive
	}

	if err := uc.repo.Update(ctx, source); err != nil {
		return response.SourceResponse{}, fmt.Errorf("updating source: %w", err)
	}

	return response.FromSource(source), nil
}
