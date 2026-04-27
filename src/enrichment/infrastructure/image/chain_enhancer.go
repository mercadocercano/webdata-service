package image

import (
	"context"
	"errors"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

// ChainEnhancer intenta el primer enhancer y cae al segundo si el primero falla.
// Usado para Replicate → Passthrough.
type ChainEnhancer struct {
	primary  *ReplicateEnhancer
	fallback *PassthroughEnhancer
}

func NewChainEnhancer(primary *ReplicateEnhancer, fallback *PassthroughEnhancer) *ChainEnhancer {
	return &ChainEnhancer{primary: primary, fallback: fallback}
}

func (c *ChainEnhancer) Enhance(ctx context.Context, rawURL string, source entity.Source) ([]byte, string, int, error) {
	quality := ClassifyBySource(source)

	if quality == QualityDirty {
		imageBytes, mimeType, costCents, err := c.primary.Enhance(ctx, rawURL, source)
		if err == nil {
			return imageBytes, mimeType, costCents, nil
		}
		// Si es ErrNotConfigured u otro error, caemos al passthrough
		if !errors.Is(err, ErrNotConfigured) {
			// Error inesperado de Replicate, igual usamos fallback
			_ = err
		}
	}

	return c.fallback.Enhance(ctx, rawURL, source)
}
