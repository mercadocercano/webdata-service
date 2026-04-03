package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
)

type ProductResponse struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	SourceID           uuid.UUID       `json:"source_id"`
	JobID              *uuid.UUID      `json:"job_id,omitempty"`
	Title              string          `json:"title"`
	Price              *float64        `json:"price,omitempty"`
	Currency           string          `json:"currency"`
	OriginalPrice      *float64        `json:"original_price,omitempty"`
	URL                string          `json:"url"`
	ImageURL           string          `json:"image_url,omitempty"`
	Description        string          `json:"description,omitempty"`
	Brand              string          `json:"brand,omitempty"`
	Category           string          `json:"category,omitempty"`
	SKU                string          `json:"sku,omitempty"`
	EAN                string          `json:"ean,omitempty"`
	InStock            bool            `json:"in_stock"`
	NormalizedCategory string          `json:"normalized_category,omitempty"`
	ConfidenceScore    *float64        `json:"confidence_score,omitempty"`
	ContentHash        string          `json:"content_hash"`
	FirstSeenAt        time.Time       `json:"first_seen_at"`
	LastSeenAt         time.Time       `json:"last_seen_at"`
	PriceChangedAt     *time.Time      `json:"price_changed_at,omitempty"`
	RawData            json.RawMessage `json:"raw_data,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func FromProduct(p *entity.ScrapedProduct) ProductResponse {
	return ProductResponse{
		ID:                 p.ID,
		TenantID:           p.TenantID,
		SourceID:           p.SourceID,
		JobID:              p.JobID,
		Title:              p.Title,
		Price:              p.Price,
		Currency:           p.Currency,
		OriginalPrice:      p.OriginalPrice,
		URL:                p.URL,
		ImageURL:           p.ImageURL,
		Description:        p.Description,
		Brand:              p.Brand,
		Category:           p.Category,
		SKU:                p.SKU,
		EAN:                p.EAN,
		InStock:            p.InStock,
		NormalizedCategory: p.NormalizedCategory,
		ConfidenceScore:    p.ConfidenceScore,
		ContentHash:        p.ContentHash.String(),
		FirstSeenAt:        p.FirstSeenAt,
		LastSeenAt:         p.LastSeenAt,
		PriceChangedAt:     p.PriceChangedAt,
		RawData:            p.RawData,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
