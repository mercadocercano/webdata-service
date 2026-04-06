package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	productrequest "github.com/mercadocercano/webdata-service/src/product/application/request"
	productusecase "github.com/mercadocercano/webdata-service/src/product/application/usecase"
	vo "github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type ProductController struct {
	listUC          *productusecase.ListProductsUseCase
	getUC           *productusecase.GetProductUseCase
	priceHistUC     *productusecase.GetPriceHistoryUseCase
	deleteUC        *productusecase.DeleteProductUseCase
	bulkDeleteUC    *productusecase.BulkDeleteProductsUseCase
	updateUC        *productusecase.UpdateProductUseCase
	assignBTUC      *productusecase.AssignBusinessTypesUseCase
	removeBTUC      *productusecase.RemoveBusinessTypeUseCase
	bulkAssignBTUC  *productusecase.BulkAssignBusinessTypeUseCase
	autoMatchBTUC   *productusecase.AutoMatchBusinessTypesUseCase
}

func NewProductController(
	listUC *productusecase.ListProductsUseCase,
	getUC *productusecase.GetProductUseCase,
	priceHistUC *productusecase.GetPriceHistoryUseCase,
	deleteUC *productusecase.DeleteProductUseCase,
	bulkDeleteUC *productusecase.BulkDeleteProductsUseCase,
	updateUC *productusecase.UpdateProductUseCase,
	assignBTUC *productusecase.AssignBusinessTypesUseCase,
	removeBTUC *productusecase.RemoveBusinessTypeUseCase,
	bulkAssignBTUC *productusecase.BulkAssignBusinessTypeUseCase,
	autoMatchBTUC *productusecase.AutoMatchBusinessTypesUseCase,
) *ProductController {
	return &ProductController{
		listUC: listUC, getUC: getUC, priceHistUC: priceHistUC,
		deleteUC: deleteUC, bulkDeleteUC: bulkDeleteUC, updateUC: updateUC,
		assignBTUC: assignBTUC, removeBTUC: removeBTUC, bulkAssignBTUC: bulkAssignBTUC,
		autoMatchBTUC: autoMatchBTUC,
	}
}

func (c *ProductController) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	req := productrequest.ListProductsRequest{
		Category:         r.URL.Query().Get("category"),
		Brand:            r.URL.Query().Get("brand"),
		BusinessTypeCode: r.URL.Query().Get("business_type"),
		Query:            r.URL.Query().Get("q"),
		SortBy:           r.URL.Query().Get("sort_by"),
		SortOrder:        r.URL.Query().Get("sort_order"),
		Page:             parseIntQuery(r, "page", 1),
		PageSize:         parseIntQuery(r, "page_size", 20),
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

func (c *ProductController) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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

	if err := c.deleteUC.Execute(r.Context(), tenantID, id); err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *ProductController) BulkDeleteProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(body.IDs) == 0 {
		middleware.JSONError(w, http.StatusBadRequest, "ids array is required")
		return
	}

	ids := make([]uuid.UUID, 0, len(body.IDs))
	for _, raw := range body.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			middleware.JSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid UUID: %s", raw))
			return
		}
		ids = append(ids, id)
	}

	deleted, err := c.bulkDeleteUC.Execute(r.Context(), tenantID, ids)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]int64{"deleted": deleted})
}

func (c *ProductController) PatchProduct(w http.ResponseWriter, r *http.Request) {
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

	var body struct {
		IsBlocked *bool `json:"is_blocked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.IsBlocked == nil {
		middleware.JSONError(w, http.StatusBadRequest, "is_blocked field is required")
		return
	}

	if err := c.updateUC.Execute(r.Context(), tenantID, id, *body.IsBlocked); err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *ProductController) AssignBusinessTypes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var body struct {
		BusinessTypes []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"business_types"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(body.BusinessTypes) == 0 {
		middleware.JSONError(w, http.StatusBadRequest, "business_types array is required")
		return
	}

	assignments := make([]vo.BusinessTypeAssignment, 0, len(body.BusinessTypes))
	for _, bt := range body.BusinessTypes {
		a, err := vo.NewBusinessTypeAssignment(bt.Code, bt.Name)
		if err != nil {
			middleware.JSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		assignments = append(assignments, a)
	}

	if err := c.assignBTUC.Execute(r.Context(), tenantID, productID, assignments); err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *ProductController) RemoveBusinessType(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		middleware.JSONError(w, http.StatusBadRequest, "business type code is required")
		return
	}

	if err := c.removeBTUC.Execute(r.Context(), tenantID, productID, code); err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *ProductController) BulkAssignBusinessType(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	var body struct {
		ProductIDs []string `json:"product_ids"`
		Code       string   `json:"code"`
		Name       string   `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(body.ProductIDs) == 0 {
		middleware.JSONError(w, http.StatusBadRequest, "product_ids array is required")
		return
	}
	if body.Code == "" {
		middleware.JSONError(w, http.StatusBadRequest, "code is required")
		return
	}

	ids := make([]uuid.UUID, 0, len(body.ProductIDs))
	for _, raw := range body.ProductIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			middleware.JSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid UUID: %s", raw))
			return
		}
		ids = append(ids, id)
	}

	assignment, err := vo.NewBusinessTypeAssignment(body.Code, body.Name)
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := c.bulkAssignBTUC.Execute(r.Context(), tenantID, ids, assignment)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, result)
}

func (c *ProductController) AutoMatchBusinessTypes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	var body struct {
		ProductIDs             []string `json:"product_ids"`
		IncludeAlreadyAssigned bool     `json:"include_already_assigned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var productIDs []uuid.UUID
	for _, raw := range body.ProductIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			middleware.JSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid UUID: %s", raw))
			return
		}
		productIDs = append(productIDs, id)
	}

	req := productusecase.AutoMatchRequest{
		ProductIDs:             productIDs,
		IncludeAlreadyAssigned: body.IncludeAlreadyAssigned,
	}

	result, err := c.autoMatchBTUC.Execute(r.Context(), tenantID, req)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, result)
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
