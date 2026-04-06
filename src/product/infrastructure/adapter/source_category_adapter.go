package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SourceCategoryAdapter implements port.SourceCategoryFinder with a lightweight
// query that only reads the category column from webdata_sources.
type SourceCategoryAdapter struct {
	db *sql.DB
}

func NewSourceCategoryAdapter(db *sql.DB) *SourceCategoryAdapter {
	return &SourceCategoryAdapter{db: db}
}

func (a *SourceCategoryAdapter) FindCategoryBySourceID(ctx context.Context, tenantID, sourceID uuid.UUID) (string, error) {
	var category string
	err := a.db.QueryRowContext(ctx,
		"SELECT category FROM webdata_sources WHERE id = $1 AND tenant_id = $2",
		sourceID, tenantID,
	).Scan(&category)
	if err != nil {
		return "", fmt.Errorf("source category not found: %w", err)
	}
	return category, nil
}
