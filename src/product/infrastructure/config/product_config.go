package config

import (
	"database/sql"

	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	productpersistence "github.com/mercadocercano/webdata-service/src/product/infrastructure/persistence"
	productadapter "github.com/mercadocercano/webdata-service/src/product/infrastructure/adapter"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
	productport "github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type ProductModule struct {
	Repo       productport.ProductRepository
	UpsertUC   *productusecase.UpsertProductsUseCase
	Controller *productcontroller.ProductController
}

func NewProductModule(db *sql.DB) *ProductModule {
	repo := productpersistence.NewPostgresProductRepository(db)
	categoryFinder := productadapter.NewSourceCategoryAdapter(db)
	upsertUC := productusecase.NewUpsertProductsUseCase(repo, categoryFinder)
	pimSyncer := productadapter.NewPIMCatalogSyncerAdapter()
	syncToPIMUC := productusecase.NewSyncProductToPIMUseCase(repo, pimSyncer)
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
		UpsertUC:   upsertUC,
		Controller: ctrl,
	}
}
