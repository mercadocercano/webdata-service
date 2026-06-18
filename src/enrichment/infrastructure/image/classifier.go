package image

import "github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"

type Quality string

const (
	QualityClean Quality = "clean"
	QualityDirty Quality = "dirty"
)

// ClassifyBySource determina si necesita BG removal.
// ML images ya vienen con fondo blanco → clean.
// DALL-E/gpt-image-1 son sintéticas, ya generadas con fondo blanco → clean (no bg-removal).
// OFF y Firecrawl son crowdsourced → dirty.
func ClassifyBySource(source entity.Source) Quality {
	if source == entity.SourceML || source == entity.SourceDALLE {
		return QualityClean
	}
	return QualityDirty
}
