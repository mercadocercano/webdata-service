package controller

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	productrequest "github.com/mercadocercano/webdata-service/src/product/application/request"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type ProductController struct {
	listUC       *productusecase.ListProductsUseCase
	getUC        *productusecase.GetProductUseCase
	priceHistUC  *productusecase.GetPriceHistoryUseCase
}

func NewProductController(
	listUC *productusecase.ListProductsUseCase,
	getUC *productusecase.GetProductUseCase,
	priceHistUC *productusecase.GetPriceHistoryUseCase,
) *ProductController {
	return &ProductController{listUC: listUC, getUC: getUC, priceHistUC: priceHistUC}
}

func (c *ProductController) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	req := productrequest.ListProductsRequest{
		Category:  r.URL.Query().Get("category"),
		Brand:     r.URL.Query().Get("brand"),
		Query:     r.URL.Query().Get("q"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
		Page:      parseIntQuery(r, "page", 1),
		PageSize:  parseIntQuery(r, "page_size", 20),
	}

	if srcID := r.URL.Query().Get("source_id"); srcID != "" {
		if id, err := uuid.Parse(srcID); err == nil {
			req.SourceID = &id
		}
	}

	result, err := c.listUC.Execute(r.Context(), tenantID, req)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, middleware.PaginatedResponse{
		Data: result.Items,
		Meta: middleware.MetaResponse{
			Page: result.Page, PageSize: result.PageSize,
			Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (c *ProductController) GetProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	resp, err := c.getUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}

func (c *ProductController) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	records, err := c.priceHistUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": records})
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return defaultVal
	}
	return n
}
