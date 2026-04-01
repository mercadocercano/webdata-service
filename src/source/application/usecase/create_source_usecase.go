package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/request"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type CreateSourceUseCase struct {
	repo port.SourceRepository
}

func NewCreateSourceUseCase(repo port.SourceRepository) *CreateSourceUseCase {
	return &CreateSourceUseCase{repo: repo}
}

func (uc *CreateSourceUseCase) Execute(ctx context.Context, tenantID uuid.UUID, req request.CreateSourceRequest) (response.SourceResponse, error) {
	existing, total, err := uc.repo.FindAll(ctx, tenantID, port.SourceFilter{})
	if err != nil {
		return response.SourceResponse{}, fmt.Errorf("checking duplicate name: %w", err)
	}
	for _, s := range existing {
		if s.Name == req.Name {
			return response.SourceResponse{}, fmt.Errorf("source with name %q already exists", req.Name)
		}
	}
	_ = total

	source, err := entity.NewSource(entity.CreateSourceParams{
		TenantID:         tenantID,
		Name:             req.Name,
		BaseURL:          req.BaseURL,
		Category:         req.Category,
		SourceType:       req.SourceType,
		City:             req.City,
		Priority:         req.Priority,
		Tier:             req.Tier,
		FirecrawlMethod:  req.FirecrawlMethod,
		CronExpression:   req.CronExpression,
		Notes:            req.Notes,
		ExtractionSchema: req.ExtractionSchema,
		CrawlConfigRaw:   req.CrawlConfigRaw,
	})
	if err != nil {
		return response.SourceResponse{}, fmt.Errorf("creating source: %w", err)
	}

	if err := uc.repo.Save(ctx, source); err != nil {
		return response.SourceResponse{}, fmt.Errorf("saving source: %w", err)
	}

	return response.FromSource(source), nil
}
