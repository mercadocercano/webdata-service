package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SourceCategoryAdapter implements port.SourceCategoryFinder with a lightweight
// query that reads the category and authoritative_category columns from webdata_sources.
type SourceCategoryAdapter struct {
	db *sql.DB
}

func NewSourceCategoryAdapter(db *sql.DB) *SourceCategoryAdapter {
	return &SourceCategoryAdapter{db: db}
}

func (a *SourceCategoryAdapter) FindCategoryBySourceID(ctx context.Context, tenantID, sourceID uuid.UUID) (string, bool, error) {
	var category string
	var authoritative bool
	err := a.db.QueryRowContext(ctx,
		"SELECT category, authoritative_category FROM webdata_sources WHERE id = $1 AND tenant_id = $2",
		sourceID, tenantID,
	).Scan(&category, &authoritative)
	if err != nil {
		return "", false, fmt.Errorf("source category not found: %w", err)
	}
	return category, authoritative, nil
}
