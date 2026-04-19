package config

import (
	statscontroller "github.com/mercadocercano/webdata-service/src/stats/infrastructure/controller"
	statsusecase "github.com/mercadocercano/webdata-service/src/stats/application/usecase"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type StatsModule struct {
	Controller *statscontroller.StatsController
}

func NewStatsModule(
	sourceRepo sourceport.SourceRepository,
	jobRepo scrapingport.ScrapingJobRepository,
	productRepo productport.ProductRepository,
) *StatsModule {
	statsUC := statsusecase.NewGetStatsUseCase(sourceRepo, jobRepo, productRepo)
	sourceStatsUC := statsusecase.NewGetSourceStatsUseCase(sourceRepo)
	pipelineStatusUC := statsusecase.NewGetPipelineStatusUseCase(sourceRepo, jobRepo, productRepo)
	ctrl := statscontroller.NewStatsController(statsUC, sourceStatsUC, pipelineStatusUC)

	return &StatsModule{Controller: ctrl}
}
