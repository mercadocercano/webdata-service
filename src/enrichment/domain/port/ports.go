package port

import (
	"context"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

// ProductSource busca datos de un producto por GTIN o nombre.
type ProductSource interface {
	FindByGTIN(ctx context.Context, gtin string) (*entity.ProductData, error)
	FindByName(ctx context.Context, name string) (*entity.ProductData, error)
	SourceName() entity.Source
}

// ImageEnhancer procesa una imagen y retorna bytes procesados.
type ImageEnhancer interface {
	Enhance(ctx context.Context, rawURL string, source entity.Source) (imageBytes []byte, mimeType string, costCents int, err error)
}

// ImageStore guarda imagen en CDN.
type ImageStore interface {
	Upload(ctx context.Context, gtin string, imageBytes []byte, mimeType string) (cdnURL string, cdnKey string, err error)
	Exists(ctx context.Context, gtin string) (bool, string, error)
}

// EnrichmentRepository tracking idempotente.
type EnrichmentRepository interface {
	IsResolved(ctx context.Context, gtin string) (bool, error)
	Save(ctx context.Context, resolution *entity.ImageResolution) error
	GetStatus(ctx context.Context) (*EnrichmentStatus, error)
}

type EnrichmentStatus struct {
	TotalResolved   int            `json:"total_resolved"`
	TotalCostCents  int            `json:"total_cost_cents"`
	BySource        map[string]int `json:"by_source"`
	LowQualityCount int            `json:"low_quality_count"`
}

// PIMClient para leer y actualizar global_products.
type PIMClient interface {
	ListNeedingEnrichment(ctx context.Context, businessType string, limit int) (*NeedsEnrichmentResult, error)
	ListNeedingEnrichmentPage(ctx context.Context, businessType string, limit, offset int) (*NeedsEnrichmentResult, error)
	UpdateProduct(ctx context.Context, productID string, req UpdateProductRequest) error
}

type NeedsEnrichmentItem struct {
	ID           string `json:"id"`
	EAN          string `json:"ean"`
	Name         string `json:"name"`
	Brand        string `json:"brand"`
	Category     string `json:"category"`
	BusinessType string `json:"business_type"`
	QualityScore int    `json:"quality_score"`
}

type NeedsEnrichmentResult struct {
	Items []NeedsEnrichmentItem `json:"products"`
	Total int                   `json:"total"`
}

type UpdateProductRequest struct {
	GTIN     string `json:"gtin,omitempty"`
	EANCode  string `json:"ean_code,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Brand    string `json:"brand,omitempty"`
	Category string `json:"category,omitempty"`
}
