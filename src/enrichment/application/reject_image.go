package application

import (
	"context"
	"fmt"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

type RejectImageRequest struct {
	ProductID string `json:"product_id"`
}

type RejectImageResult struct {
	ProductID string `json:"product_id"`
	Cleared   bool   `json:"cleared"`
}

type RejectImageUseCase struct {
	repo port.EnrichmentRepository
	pim  port.PIMClient
}

func NewRejectImageUseCase(repo port.EnrichmentRepository, pim port.PIMClient) *RejectImageUseCase {
	return &RejectImageUseCase{repo: repo, pim: pim}
}

func (uc *RejectImageUseCase) Execute(ctx context.Context, req RejectImageRequest) (*RejectImageResult, error) {
	if req.ProductID == "" {
		return nil, fmt.Errorf("product_id es requerido")
	}

	currentURL, err := uc.repo.GetCurrentRawURL(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("obteniendo imagen actual: %w", err)
	}

	if err := uc.repo.RejectImage(ctx, req.ProductID, currentURL); err != nil {
		return nil, fmt.Errorf("rechazando imagen: %w", err)
	}

	empty := ""
	if err := uc.pim.UpdateProduct(ctx, req.ProductID, port.UpdateProductRequest{ImageURL: &empty}); err != nil {
		return nil, fmt.Errorf("limpiando imagen en PIM: %w", err)
	}

	return &RejectImageResult{
		ProductID: req.ProductID,
		Cleared:   true,
	}, nil
}
