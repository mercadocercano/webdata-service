package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/request"
	"github.com/mercadocercano/webdata-service/src/product/application/response"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type ListProductsResult struct {
	Items      []response.ProductResponse
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

type ListProductsUseCase struct {
	repo port.ProductRepository
}

func NewListProductsUseCase(repo port.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{repo: repo}
}

func (uc *ListProductsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, req request.ListProductsRequest) (ListProductsResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	filter := port.ProductFilter{
		SourceID:           req.SourceID,
		Category:           req.Category,
		NormalizedCategory: req.NormalizedCategory,
		Brand:              req.Brand,
		BusinessTypeCode:   req.BusinessTypeCode,
		MinPrice:           req.MinPrice,
		MaxPrice:           req.MaxPrice,
		Query:              req.Query,
		SortBy:             req.SortBy,
		SortOrder:          req.SortOrder,
		Page:               req.Page,
		PageSize:           req.PageSize,
		WithoutImage:       req.WithoutImage,
		NotSyncedToPIM:     req.NotSyncedToPIM,
	}

	products, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListProductsResult{}, err
	}

	items := make([]response.ProductResponse, len(products))
	for i, p := range products {
		items[i] = response.FromProduct(p)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return ListProductsResult{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}
