package controller

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type JobController struct {
	listUC   *scrapingusecase.ListJobsUseCase
	getUC    *scrapingusecase.GetJobUseCase
	cancelUC *scrapingusecase.CancelJobUseCase
	retryUC  *scrapingusecase.RetryJobUseCase
}

func NewJobController(
	listUC *scrapingusecase.ListJobsUseCase,
	getUC *scrapingusecase.GetJobUseCase,
	cancelUC *scrapingusecase.CancelJobUseCase,
	retryUC *scrapingusecase.RetryJobUseCase,
) *JobController {
	return &JobController{listUC: listUC, getUC: getUC, cancelUC: cancelUC, retryUC: retryUC}
}

func (c *JobController) ListJobs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	filter := scrapingport.JobFilter{
		Status:   r.URL.Query().Get("status"),
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
			Page: result.Page, PageSize: result.PageSize,
			Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (c *JobController) GetJob(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	resp, err := c.getUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		middleware.JSONError(w, http.StatusNotFound, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}

func (c *JobController) CancelJob(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := c.cancelUC.Execute(r.Context(), tenantID, id); err != nil {
		middleware.JSONError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *JobController) RetryJob(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	resp, err := c.retryUC.Execute(r.Context(), tenantID, id)
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
