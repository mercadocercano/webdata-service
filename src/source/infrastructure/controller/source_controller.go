package controller

import (
	"fmt"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	sourcerequest "github.com/mercadocercano/webdata-service/src/source/application/request"
	sourceusecase "github.com/mercadocercano/webdata-service/src/source/application/usecase"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type SourceController struct {
	createUC  *sourceusecase.CreateSourceUseCase
	getUC     *sourceusecase.GetSourceUseCase
	listUC    *sourceusecase.ListSourcesUseCase
	updateUC  *sourceusecase.UpdateSourceUseCase
	deleteUC  *sourceusecase.DeleteSourceUseCase
	triggerUC *scrapingusecase.TriggerScrapeUseCase
}

func NewSourceController(
	createUC *sourceusecase.CreateSourceUseCase,
	getUC *sourceusecase.GetSourceUseCase,
	listUC *sourceusecase.ListSourcesUseCase,
	updateUC *sourceusecase.UpdateSourceUseCase,
	deleteUC *sourceusecase.DeleteSourceUseCase,
	triggerUC *scrapingusecase.TriggerScrapeUseCase,
) *SourceController {
	return &SourceController{
		createUC:  createUC,
		getUC:     getUC,
		listUC:    listUC,
		updateUC:  updateUC,
		deleteUC:  deleteUC,
		triggerUC: triggerUC,
	}
}

func (c *SourceController) CreateSource(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	var req sourcerequest.CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := c.createUC.Execute(r.Context(), tenantID, req)
	if err != nil {
		middleware.JSONError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusCreated, map[string]interface{}{"data": resp})
}

func (c *SourceController) GetSource(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid source ID")
		return
	}

	resp, err := c.getUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}

func (c *SourceController) ListSources(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	filter := sourceport.SourceFilter{
		Category: r.URL.Query().Get("category"),
		Page:     parseIntQuery(r, "page", 1),
		PageSize: parseIntQuery(r, "page_size", 20),
	}

	result, err := c.listUC.Execute(r.Context(), tenantID, filter)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, middleware.PaginatedResponse{
		Data: result.Items,
		Meta: middleware.MetaResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (c *SourceController) UpdateSource(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid source ID")
		return
	}

	var req sourcerequest.UpdateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := c.updateUC.Execute(r.Context(), tenantID, id, req)
	if err != nil {
		middleware.JSONError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}

func (c *SourceController) DeleteSource(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid source ID")
		return
	}

	if err := c.deleteUC.Execute(r.Context(), tenantID, id); err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *SourceController) TriggerScrape(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	sourceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid source ID")
		return
	}

	resp, err := c.triggerUC.Execute(r.Context(), tenantID, sourceID)
	if err != nil {
		middleware.JSONError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusCreated, map[string]interface{}{"data": resp})
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
