package controller

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/application/request"
	"github.com/mercadocercano/webdata-service/src/product/application/usecase"
	"github.com/mercadocercano/webdata-service/src/shared/httputil"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type ProductController struct {
	listUC       *usecase.ListProductsUseCase
	getUC        *usecase.GetProductUseCase
	priceHistUC  *usecase.GetPriceHistoryUseCase
}

func NewProductController(
	listUC *usecase.ListProductsUseCase,
	getUC *usecase.GetProductUseCase,
	priceHistUC *usecase.GetPriceHistoryUseCase,
) *ProductController {
	return &ProductController{listUC: listUC, getUC: getUC, priceHistUC: priceHistUC}
}

func (c *ProductController) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	q := r.URL.Query()
	req := request.ListProductsRequest{
		Category:  q.Get("category"),
		Brand:     q.Get("brand"),
		Query:     q.Get("q"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}
	if v := q.Get("source_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			req.SourceID = &id
		}
	}
	if v := q.Get("min_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			req.MinPrice = &f
		}
	}
	if v := q.Get("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			req.MaxPrice = &f
		}
	}
	req.Page, _ = strconv.Atoi(q.Get("page"))
	req.PageSize, _ = strconv.Atoi(q.Get("page_size"))

	result, err := c.listUC.Execute(r.Context(), tenantID, req)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.Paginated(w, http.StatusOK, result.Products, result.Page, result.PageSize, result.Total)
}

func (c *ProductController) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	resp, err := c.getUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (c *ProductController) PriceHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	history, err := c.priceHistUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, history)
}
