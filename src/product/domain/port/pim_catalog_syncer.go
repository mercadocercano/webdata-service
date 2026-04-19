package port

import (
	"context"

	"github.com/google/uuid"
)

// NeedsEnrichmentItem representa un producto de global_products con datos incompletos.
type NeedsEnrichmentItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Brand        string `json:"brand"`
	BusinessType string `json:"business_type"`
	Category     string `json:"category"`
	QualityScore int    `json:"quality_score"`
}

// ListNeedingEnrichmentResult es la respuesta del endpoint needs-enrichment de PIM.
type ListNeedingEnrichmentResult struct {
	Items []NeedsEnrichmentItem `json:"items"`
	Total int                   `json:"total"`
}

type PIMGlobalProduct struct {
	ID           uuid.UUID `json:"id"`
	EAN          string    `json:"ean"`
	Name         string    `json:"name"`
	Brand        string    `json:"brand"`
	Category     string    `json:"category"`
	Price        *float64  `json:"price"`
	ImageURL     string    `json:"image_url"`
	Source       string    `json:"source"`
	SourceURL    string    `json:"source_url"`
	BusinessType string    `json:"business_type"`
}

type CreatePIMProductRequest struct {
	EAN          string   `json:"ean,omitempty"`
	Name         string   `json:"name"`
	Brand        string   `json:"brand,omitempty"`
	Category     string   `json:"category,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	ImageURL     string   `json:"image_url,omitempty"`
	Source       string   `json:"source"`
	SourceURL    string   `json:"source_url,omitempty"`
	BusinessType string   `json:"business_type,omitempty"`
}

type UpdatePIMProductRequest struct {
	Price    *float64 `json:"price,omitempty"`
	ImageURL string   `json:"image_url,omitempty"`
}

type PIMCatalogSyncer interface {
	SearchByEAN(ctx context.Context, tenantID uuid.UUID, ean string) (*PIMGlobalProduct, error)
	SearchByNameBrand(ctx context.Context, tenantID uuid.UUID, name, brand string) (*PIMGlobalProduct, error)
	Create(ctx context.Context, tenantID uuid.UUID, req CreatePIMProductRequest) (*PIMGlobalProduct, error)
	Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdatePIMProductRequest) error
	RefreshTemplateProducts(ctx context.Context) error
	ListNeedingEnrichment(ctx context.Context, businessType string, limit, offset int) (*ListNeedingEnrichmentResult, error)
}
