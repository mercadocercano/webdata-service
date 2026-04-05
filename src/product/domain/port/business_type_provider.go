package port

import "context"

// BusinessType represents a business type fetched from pim-service.
type BusinessType struct {
	Code string
	Name string
}

// BusinessTypeProvider fetches available business types from an external source.
type BusinessTypeProvider interface {
	FetchAll(ctx context.Context, tenantID string) ([]BusinessType, error)
}
