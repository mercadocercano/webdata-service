package config

import (
	"database/sql"

	"github.com/mercadocercano/webdata-service/src/enrichment/application"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
	enrichcontroller "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/controller"
	enrichimage "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/image"
	enrichpersistence "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/persistence"
	enrichpim "github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/pim"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/dalle"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/firecrawl"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/mercadolibre"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/openfoodfacts"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/perplexity"
)

type EnrichmentModule struct {
	Handler *enrichcontroller.EnrichmentHandler
}

func NewEnrichmentModule(db *sql.DB) *EnrichmentModule {
	// Sources en cascada: ML → OFF → Firecrawl → Perplexity → DALL-E
	mlAdapter := mercadolibre.NewMLAdapter()
	sources := []port.ProductSource{
		mlAdapter,
		openfoodfacts.NewOFFAdapter(),
		firecrawl.NewFirecrawlEnrichAdapter(),
		perplexity.NewPerplexityGTINSource(mlAdapter),
		dalle.NewDALLEGeneratorSource(),
	}

	// Enhancers: Replicate con fallback a Passthrough
	replicateEnhancer := enrichimage.NewReplicateEnhancer()
	passthroughEnhancer := enrichimage.NewPassthroughEnhancer()
	enhancer := enrichimage.NewChainEnhancer(replicateEnhancer, passthroughEnhancer)

	store := enrichimage.NewSpacesStore()
	repo := enrichpersistence.NewPostgresEnrichmentRepository(db)
	pimClient := enrichpim.NewPIMClientAdapter()

	runBatchUC := application.NewRunEnrichmentBatchUseCase(pimClient, sources, enhancer, store, repo)
	getStatusUC := application.NewGetEnrichmentStatusUseCase(repo)

	handler := enrichcontroller.NewEnrichmentHandler(runBatchUC, getStatusUC)

	return &EnrichmentModule{Handler: handler}
}
