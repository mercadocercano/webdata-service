package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/request"
	"github.com/mercadocercano/webdata-service/src/product/application/response"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type ListProductsUseCase struct {
	repo port.ProductRepository
}

func NewListProductsUseCase(repo port.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{repo: repo}
}

type ListProductsResult struct {
	Products []response.ProductResponse
	Total    int
	Page     int
	PageSize int
}

func (uc *ListProductsUseCase) Execute(ctx context.Context, tenantID uuid.UUID, req request.ListProductsRequest) (ListProductsResult, error) {
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	filter := port.ProductFilter{
		SourceID:           req.SourceID,
		Category:           req.Category,
		NormalizedCategory: req.NormalizedCategory,
		Brand:              req.Brand,
		MinPrice:           req.MinPrice,
		MaxPrice:           req.MaxPrice,
		Query:              req.Query,
		SortBy:             req.SortBy,
		SortOrder:          req.SortOrder,
		Page:               req.Page,
		PageSize:           req.PageSize,
	}

	products, total, err := uc.repo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return ListProductsResult{}, fmt.Errorf("listing products: %w", err)
	}

	result := make([]response.ProductResponse, len(products))
	for i, p := range products {
		result[i] = response.FromProduct(p)
	}

	return ListProductsResult{Products: result, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}
