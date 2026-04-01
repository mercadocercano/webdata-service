package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	srcusecase "github.com/mercadocercano/webdata-service/src/source/application/usecase"
	"github.com/mercadocercano/webdata-service/src/source/application/request"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/httputil"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type SourceController struct {
	createUC  *srcusecase.CreateSourceUseCase
	updateUC  *srcusecase.UpdateSourceUseCase
	listUC    *srcusecase.ListSourcesUseCase
	getUC     *srcusecase.GetSourceUseCase
	deleteUC  *srcusecase.DeleteSourceUseCase
	triggerUC *usecase.TriggerScrapeUseCase
}

func NewSourceController(
	createUC *srcusecase.CreateSourceUseCase,
	updateUC *srcusecase.UpdateSourceUseCase,
	listUC *srcusecase.ListSourcesUseCase,
	getUC *srcusecase.GetSourceUseCase,
	deleteUC *srcusecase.DeleteSourceUseCase,
	triggerUC *usecase.TriggerScrapeUseCase,
) *SourceController {
	return &SourceController{
		createUC:  createUC,
		updateUC:  updateUC,
		listUC:    listUC,
		getUC:     getUC,
		deleteUC:  deleteUC,
		triggerUC: triggerUC,
	}
}

func (c *SourceController) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	var req request.CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := c.createUC.Execute(r.Context(), tenantID, req)
	if err != nil {
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httputil.JSON(w, http.StatusCreated, resp)
}

func (c *SourceController) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	q := r.URL.Query()
	filter := port.SourceFilter{
		Category: q.Get("category"),
	}
	if v := q.Get("is_active"); v != "" {
		b, _ := strconv.ParseBool(v)
		filter.IsActive = &b
	}
	filter.Page, _ = strconv.Atoi(q.Get("page"))
	filter.PageSize, _ = strconv.Atoi(q.Get("page_size"))

	result, err := c.listUC.Execute(r.Context(), tenantID, filter)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.Paginated(w, http.StatusOK, result.Sources, result.Page, result.PageSize, result.Total)
}

func (c *SourceController) Get(w http.ResponseWriter, r *http.Request) {
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

func (c *SourceController) Update(w http.ResponseWriter, r *http.Request) {
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

	var req request.UpdateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := c.updateUC.Execute(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (c *SourceController) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := c.deleteUC.Execute(r.Context(), tenantID, id); err != nil {
		httputil.Error(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *SourceController) Trigger(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	sourceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	resp, err := c.triggerUC.Execute(r.Context(), tenantID, sourceID)
	if err != nil {
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httputil.JSON(w, http.StatusCreated, resp)
}
