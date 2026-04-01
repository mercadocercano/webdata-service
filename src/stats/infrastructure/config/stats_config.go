package config

import (
	statscontroller "github.com/mercadocercano/webdata-service/src/stats/infrastructure/controller"
	statsuc "github.com/mercadocercano/webdata-service/src/stats/application/usecase"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type StatsModule struct {
	Controller *statscontroller.StatsController
}

func NewStatsModule(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapeport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *StatsModule {
	getStatsUC := statsuc.NewGetStatsUseCase(sourceRepo, jobRepo, productRepo)
	getSourceStatsUC := statsuc.NewGetSourceStatsUseCase(sourceRepo, jobRepo, productRepo)

	return &StatsModule{
		Controller: statscontroller.NewStatsController(getStatsUC, getSourceStatsUC),
	}
}
