package config

import (
	"database/sql"

	prdcontroller "github.com/mercadocercano/webdata-service/src/product/infrastructure/controller"
	prdpersistence "github.com/mercadocercano/webdata-service/src/product/infrastructure/persistence"
	prduc "github.com/mercadocercano/webdata-service/src/product/application/usecase"
)

type ProductModule struct {
	Controller *prdcontroller.ProductController
	Repo       *prdpersistence.PostgresProductRepository
	UpsertUC   *prduc.UpsertProductsUseCase
}

func NewProductModule(db *sql.DB) *ProductModule {
	repo := prdpersistence.NewPostgresProductRepository(db)
	listUC := prduc.NewListProductsUseCase(repo)
	getUC := prduc.NewGetProductUseCase(repo)
	priceHistUC := prduc.NewGetPriceHistoryUseCase(repo)
	upsertUC := prduc.NewUpsertProductsUseCase(repo)

	return &ProductModule{
		Controller: prdcontroller.NewProductController(listUC, getUC, priceHistUC),
		Repo:       repo,
		UpsertUC:   upsertUC,
	}
}
