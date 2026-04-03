package config

import (
	"database/sql"

	sourcecontroller "github.com/mercadocercano/webdata-service/src/source/infrastructure/controller"
	sourcepersistence "github.com/mercadocercano/webdata-service/src/source/infrastructure/persistence"
	sourceusecase "github.com/mercadocercano/webdata-service/src/source/application/usecase"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type SourceModule struct {
	Repo       sourceport.SourceRepository
	Controller *sourcecontroller.SourceController
}

func NewSourceModule(db *sql.DB, jobRepo scrapingport.ScrapingJobRepository) *SourceModule {
	repo := sourcepersistence.NewPostgresSourceRepository(db)
	createUC := sourceusecase.NewCreateSourceUseCase(repo)
	getUC := sourceusecase.NewGetSourceUseCase(repo)
	listUC := sourceusecase.NewListSourcesUseCase(repo)
	updateUC := sourceusecase.NewUpdateSourceUseCase(repo)
	deleteUC := sourceusecase.NewDeleteSourceUseCase(repo)

	var triggerUC *scrapingusecase.TriggerScrapeUseCase
	if jobRepo != nil {
		triggerUC = scrapingusecase.NewTriggerScrapeUseCase(repo, jobRepo)
	}

	ctrl := sourcecontroller.NewSourceController(createUC, getUC, listUC, updateUC, deleteUC, triggerUC)

	return &SourceModule{
		Repo:       repo,
		Controller: ctrl,
	}
}
