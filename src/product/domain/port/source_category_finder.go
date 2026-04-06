package port

import (
	"context"

	"github.com/google/uuid"
)

// SourceCategoryFinder provides read-only access to a source's category.
// Follows Interface Segregation: UpsertProductsUseCase only needs the category,
// not the full SourceRepository.
type SourceCategoryFinder interface {
	FindCategoryBySourceID(ctx context.Context, tenantID, sourceID uuid.UUID) (string, error)
}
