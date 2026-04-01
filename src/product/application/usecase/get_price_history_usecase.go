package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/response"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type GetPriceHistoryUseCase struct {
	repo port.ProductRepository
}

func NewGetPriceHistoryUseCase(repo port.ProductRepository) *GetPriceHistoryUseCase {
	return &GetPriceHistoryUseCase{repo: repo}
}

func (uc *GetPriceHistoryUseCase) Execute(ctx context.Context, tenantID, productID uuid.UUID) ([]response.PriceHistoryResponse, error) {
	records, err := uc.repo.FindPriceHistory(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	result := make([]response.PriceHistoryResponse, len(records))
	for i, r := range records {
		result[i] = response.FromPriceRecord(r)
	}
	return result, nil
}
