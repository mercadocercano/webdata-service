package port

import (
	"context"

	"github.com/google/uuid"
)

// SourceCategoryFinder provides read-only access to a source's category and
// whether that category is authoritative (single-rubro dedicated source).
// Follows Interface Segregation: UpsertProductsUseCase only needs these fields,
// not the full SourceRepository.
//
// authoritative=true → la fuente es de un solo rubro (Easy, Puppis, Blaisten);
// el caller debe usar el mapeo de la fuente y SALTEAR el resolver por-producto.
type SourceCategoryFinder interface {
	FindCategoryBySourceID(ctx context.Context, tenantID, sourceID uuid.UUID) (category string, authoritative bool, err error)
}
