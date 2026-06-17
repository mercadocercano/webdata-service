package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/request"
	"github.com/mercadocercano/webdata-service/src/source/application/response"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/mercadocercano/webdata-service/src/source/domain/exception"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type CreateSourceUseCase struct {
	repo port.SourceRepository
}

func NewCreateSourceUseCase(repo port.SourceRepository) *CreateSourceUseCase {
	return &CreateSourceUseCase{repo: repo}
}

func (uc *CreateSourceUseCase) Execute(ctx context.Context, tenantID uuid.UUID, req request.CreateSourceRequest) (response.SourceResponse, error) {
	params := entity.CreateSourceParams{
		TenantID:        tenantID,
		Name:            req.Name,
		BaseURL:         req.BaseURL,
		Category:        req.Category,
		SourceType:      req.SourceType,
		City:            req.City,
		Priority:        req.Priority,
		Tier:            req.Tier,
		FirecrawlMethod: req.FirecrawlMethod,
		CronExpression:  req.CronExpression,
		Notes:           req.Notes,
		ExcludedBrands:  req.ExcludedBrands,
	}

	// Check for duplicate name within tenant
	existing, _, _ := uc.repo.FindAll(ctx, tenantID, port.SourceFilter{PageSize: 9999})
	for _, s := range existing {
		if s.Name == req.Name {
			return response.SourceResponse{}, exception.DuplicateSourceNameError{
				Name:     req.Name,
				TenantID: tenantID.String(),
			}
		}
	}

	source, err := entity.NewSource(params)
	if err != nil {
		return response.SourceResponse{}, err
	}

	if err := uc.repo.Save(ctx, source); err != nil {
		return response.SourceResponse{}, err
	}

	return response.FromSource(source), nil
}
