package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type ScrapedProduct struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	SourceID           uuid.UUID
	JobID              *uuid.UUID
	Title              string
	Price              *float64
	Currency           string
	OriginalPrice      *float64
	URL                string
	ImageURL           string
	Description        string
	Brand              string
	Category           string
	SKU                string
	EAN                string
	InStock            bool
	NormalizedCategory string
	ConfidenceScore    *float64
	ContentHash        value_object.ContentHash
	FirstSeenAt        time.Time
	LastSeenAt         time.Time
	PriceChangedAt     *time.Time
	RawData            json.RawMessage
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateProductParams struct {
	TenantID           uuid.UUID
	SourceID           uuid.UUID
	JobID              *uuid.UUID
	Title              string
	Price              *float64
	Currency           string
	OriginalPrice      *float64
	URL                string
	ImageURL           string
	Description        string
	Brand              string
	Category           string
	SKU                string
	EAN                string
	InStock            *bool
	NormalizedCategory string
	ConfidenceScore    *float64
	ContentHash        value_object.ContentHash
	RawData            json.RawMessage
}

func NewScrapedProduct(p CreateProductParams) (*ScrapedProduct, error) {
	if p.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	// URL is optional — extraction schemas may not always include individual product URLs.
	// Callers should provide SourceBaseURL as a fallback when URL is empty.

	currency := p.Currency
	if currency == "" {
		currency = "ARS"
	}

	inStock := true
	if p.InStock != nil {
		inStock = *p.InStock
	}

	now := time.Now()
	return &ScrapedProduct{
		ID:                 uuid.New(),
		TenantID:           p.TenantID,
		SourceID:           p.SourceID,
		JobID:              p.JobID,
		Title:              p.Title,
		Price:              p.Price,
		Currency:           currency,
		OriginalPrice:      p.OriginalPrice,
		URL:                p.URL,
		ImageURL:           p.ImageURL,
		Description:        p.Description,
		Brand:              p.Brand,
		Category:           p.Category,
		SKU:                p.SKU,
		EAN:                p.EAN,
		InStock:            inStock,
		NormalizedCategory: p.NormalizedCategory,
		ConfidenceScore:    p.ConfidenceScore,
		ContentHash:        p.ContentHash,
		FirstSeenAt:        now,
		LastSeenAt:         now,
		RawData:            p.RawData,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func (sp *ScrapedProduct) HasPriceChanged(newPrice float64) bool {
	if sp.Price == nil {
		return false
	}
	return *sp.Price != newPrice
}

func (sp *ScrapedProduct) UpdatePrice(newPrice float64) {
	now := time.Now()
	sp.Price = &newPrice
	sp.PriceChangedAt = &now
	sp.LastSeenAt = now
	sp.UpdatedAt = now
}

func (sp *ScrapedProduct) TouchLastSeen() {
	now := time.Now()
	sp.LastSeenAt = now
	sp.UpdatedAt = now
}
