package config

import (
	"database/sql"

	productcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	productpersistence "github.com/mercadocercano/webdata-service/src/product/infrastructure/persistence"
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
	upsertUC := productusecase.NewUpsertProductsUseCase(repo)
	listUC := productusecase.NewListProductsUseCase(repo)
	getUC := productusecase.NewGetProductUseCase(repo)
	priceHistUC := productusecase.NewGetPriceHistoryUseCase(repo)
	ctrl := productcontroller.NewProductController(listUC, getUC, priceHistUC)

	return &ProductModule{
		Repo:       repo,
		UpsertUC:   upsertUC,
		Controller: ctrl,
	}
}
