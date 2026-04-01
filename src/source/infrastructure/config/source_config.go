package config

import (
	"database/sql"

	scrapeuc "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	srccontroller "github.com/mercadocercano/webdata-service/src/source/infrastructure/controller"
	srcpersistence "github.com/mercadocercano/webdata-service/src/source/infrastructure/persistence"
	srcuc "github.com/mercadocercano/webdata-service/src/source/application/usecase"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

type SourceModule struct {
	Controller *srccontroller.SourceController
	Repo       *srcpersistence.PostgresSourceRepository
}

func NewSourceModule(db *sql.DB, jobRepo scrapeport.ScrapingJobRepository) *SourceModule {
	repo := srcpersistence.NewPostgresSourceRepository(db)
	createUC := srcuc.NewCreateSourceUseCase(repo)
	updateUC := srcuc.NewUpdateSourceUseCase(repo)
	listUC := srcuc.NewListSourcesUseCase(repo)
	getUC := srcuc.NewGetSourceUseCase(repo)
	deleteUC := srcuc.NewDeleteSourceUseCase(repo)
	triggerUC := scrapeuc.NewTriggerScrapeUseCase(repo, jobRepo)

	return &SourceModule{
		Controller: srccontroller.NewSourceController(createUC, updateUC, listUC, getUC, deleteUC, triggerUC),
		Repo:       repo,
	}
}
