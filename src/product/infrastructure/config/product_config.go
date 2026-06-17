package config

import (
	"database/sql"

	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
	productadapter "github.com/mercadocercano/webdata-service/src/product/infrastructure/adapter"
	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	productpersistence "github.com/mercadocercano/webdata-service/src/product/infrastructure/persistence"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	webdataport "github.com/mercadocercano/webdata-service/src/webdata/domain/port"
)

type ProductModule struct {
	Repo       productport.ProductRepository
	PIMSyncer  productport.PIMCatalogSyncer
	UpsertUC   *productusecase.UpsertProductsUseCase
	SyncUC     *productusecase.SyncProductToPIMUseCase
	Controller *productcontroller.ProductController
	logger     webdataport.WebdataEventLogger
}

// WireEnrichment inyecta el use case de enrichment una vez que sourceRepo y jobRepo están listos.
func (m *ProductModule) WireEnrichment(sourceRepo sourceport.SourceRepository, jobRepo scrapingport.ScrapingJobRepository) {
	enrichUC := productusecase.NewEnrichGlobalProductsUseCase(m.PIMSyncer, sourceRepo, jobRepo)
	if m.logger != nil {
		enrichUC.WithLogger(m.logger)
	}
	m.Controller.SetEnrichUseCase(enrichUC)
}

func NewProductModule(db *sql.DB, logger webdataport.WebdataEventLogger) *ProductModule {
	repo := productpersistence.NewPostgresProductRepository(db)
	categoryFinder := productadapter.NewSourceCategoryAdapter(db)
	upsertUC := productusecase.NewUpsertProductsUseCase(repo, categoryFinder)
	if logger != nil {
		upsertUC.WithLogger(logger)
	}
	pimSyncer := productadapter.NewPIMCatalogSyncerAdapter()
	syncToPIMUC := productusecase.NewSyncProductToPIMUseCase(repo, pimSyncer)
	if logger != nil {
		syncToPIMUC.WithLogger(logger)
	}
	upsertUC.WithPIMSync(syncToPIMUC)
	listUC := productusecase.NewListProductsUseCase(repo)
	getUC := productusecase.NewGetProductUseCase(repo)
	priceHistUC := productusecase.NewGetPriceHistoryUseCase(repo)
	deleteUC := productusecase.NewDeleteProductUseCase(repo)
	bulkDeleteUC := productusecase.NewBulkDeleteProductsUseCase(repo)
	updateUC := productusecase.NewUpdateProductUseCase(repo)
	assignBTUC := productusecase.NewAssignBusinessTypesUseCase(repo)
	removeBTUC := productusecase.NewRemoveBusinessTypeUseCase(repo)
	bulkAssignBTUC := productusecase.NewBulkAssignBusinessTypeUseCase(repo)
	btProvider := productadapter.NewPIMBusinessTypeProvider()
	autoMatchBTUC := productusecase.NewAutoMatchBusinessTypesUseCase(repo, btProvider)
	getFiltersUC := productusecase.NewGetProductFiltersUseCase(repo)
	ctrl := productcontroller.NewProductController(listUC, getUC, priceHistUC, deleteUC, bulkDeleteUC, updateUC, assignBTUC, removeBTUC, bulkAssignBTUC, autoMatchBTUC, getFiltersUC, syncToPIMUC)

	return &ProductModule{
		Repo:       repo,
		PIMSyncer:  pimSyncer,
		UpsertUC:   upsertUC,
		SyncUC:     syncToPIMUC,
		Controller: ctrl,
		logger:     logger,
	}
}
