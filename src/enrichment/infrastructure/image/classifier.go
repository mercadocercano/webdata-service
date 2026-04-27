package image

import "github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"

type Quality string

const (
	QualityClean Quality = "clean"
	QualityDirty Quality = "dirty"
)

// ClassifyBySource determina si necesita BG removal.
// ML images ya vienen con fondo blanco → clean.
// OFF y Firecrawl son crowdsourced → dirty.
func ClassifyBySource(source entity.Source) Quality {
	if source == entity.SourceML {
		return QualityClean
	}
	return QualityDirty
}
