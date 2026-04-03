package config

import (
	"database/sql"

	scrapingcontroller "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/controller"
	scrapingpersistence "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/persistence"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
)

type ScrapingModule struct {
	Repo       scrapingport.ScrapingJobRepository
	Controller *scrapingcontroller.JobController
	ExecuteUC  *scrapingusecase.ExecuteScrapingUseCase
}

func NewScrapingModule(
	db *sql.DB,
	sourceRepo sourceport.SourceRepository,
	scraperPort scrapingport.ScraperPort,
	upsertUC *productusecase.UpsertProductsUseCase,
) *ScrapingModule {
	repo := scrapingpersistence.NewPostgresJobRepository(db)
	executeUC := scrapingusecase.NewExecuteScrapingUseCase(repo, sourceRepo, scraperPort, upsertUC)
	listUC := scrapingusecase.NewListJobsUseCase(repo)
	getUC := scrapingusecase.NewGetJobUseCase(repo)
	cancelUC := scrapingusecase.NewCancelJobUseCase(repo)
	retryUC := scrapingusecase.NewRetryJobUseCase(repo)
	ctrl := scrapingcontroller.NewJobController(listUC, getUC, cancelUC, retryUC)

	return &ScrapingModule{
		Repo:       repo,
		Controller: ctrl,
		ExecuteUC:  executeUC,
	}
}
