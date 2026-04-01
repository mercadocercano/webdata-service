package config

import (
	"database/sql"

	scrapeuc "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	scrapecontroller "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/controller"
	scrapepersistence "github.com/mercadocercano/webdata-service/src/scraping/infrastructure/persistence"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

type ScrapingModule struct {
	Controller *scrapecontroller.JobController
	Repo       *scrapepersistence.PostgresJobRepository
	ExecuteUC  *scrapeuc.ExecuteScrapeUseCase
}

func NewScrapingModule(db *sql.DB, sourceRepo sourceport.SourceRepository, scraper scrapeport.ScraperPort, upserter scrapeuc.ProductUpserter) *ScrapingModule {
	repo := scrapepersistence.NewPostgresJobRepository(db)

	listUC := scrapeuc.NewListJobsUseCase(repo)
	getUC := scrapeuc.NewGetJobUseCase(repo)
	cancelUC := scrapeuc.NewCancelJobUseCase(repo)
	retryUC := scrapeuc.NewRetryJobUseCase(repo)
	executeUC := scrapeuc.NewExecuteScrapeUseCase(sourceRepo, repo, scraper, upserter)

	return &ScrapingModule{
		Controller: scrapecontroller.NewJobController(listUC, getUC, cancelUC, retryUC),
		Repo:       repo,
		ExecuteUC:  executeUC,
	}
}
