package controller

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/httputil"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type JobController struct {
	listUC   *usecase.ListJobsUseCase
	getUC    *usecase.GetJobUseCase
	cancelUC *usecase.CancelJobUseCase
	retryUC  *usecase.RetryJobUseCase
}

func NewJobController(
	listUC *usecase.ListJobsUseCase,
	getUC *usecase.GetJobUseCase,
	cancelUC *usecase.CancelJobUseCase,
	retryUC *usecase.RetryJobUseCase,
) *JobController {
	return &JobController{listUC: listUC, getUC: getUC, cancelUC: cancelUC, retryUC: retryUC}
}

func (c *JobController) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	q := r.URL.Query()
	filter := port.JobFilter{
		Status: q.Get("status"),
	}
	if v := q.Get("source_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.SourceID = &id
		}
	}
	filter.Page, _ = strconv.Atoi(q.Get("page"))
	filter.PageSize, _ = strconv.Atoi(q.Get("page_size"))

	result, err := c.listUC.Execute(r.Context(), tenantID, filter)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.Paginated(w, http.StatusOK, result.Jobs, result.Page, result.PageSize, result.Total)
}

func (c *JobController) Get(w http.ResponseWriter, r *http.Request) {
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

func (c *JobController) Cancel(w http.ResponseWriter, r *http.Request) {
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

	if err := c.cancelUC.Execute(r.Context(), tenantID, id); err != nil {
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *JobController) Retry(w http.ResponseWriter, r *http.Request) {
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

	resp, err := c.retryUC.Execute(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httputil.JSON(w, http.StatusCreated, resp)
}
