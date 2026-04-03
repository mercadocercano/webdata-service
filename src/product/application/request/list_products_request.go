package request

import "github.com/google/uuid"

type ListProductsRequest struct {
	SourceID           *uuid.UUID `json:"source_id,omitempty"`
	Category           string     `json:"category,omitempty"`
	NormalizedCategory string     `json:"normalized_category,omitempty"`
	Brand              string     `json:"brand,omitempty"`
	MinPrice           *float64   `json:"min_price,omitempty"`
	MaxPrice           *float64   `json:"max_price,omitempty"`
	Query              string     `json:"q,omitempty"`
	SortBy             string     `json:"sort_by,omitempty"`
	SortOrder          string     `json:"sort_order,omitempty"`
	Page               int        `json:"page,omitempty"`
	PageSize           int        `json:"page_size,omitempty"`
}
