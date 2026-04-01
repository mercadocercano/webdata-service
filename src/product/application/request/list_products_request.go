package request

import "github.com/google/uuid"

type ListProductsRequest struct {
	SourceID           *uuid.UUID
	Category           string
	NormalizedCategory string
	Brand              string
	MinPrice           *float64
	MaxPrice           *float64
	Query              string
	SortBy             string
	SortOrder          string
	Page               int
	PageSize           int
}
