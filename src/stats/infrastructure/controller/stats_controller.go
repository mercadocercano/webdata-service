package controller

import (
	"net/http"

	"github.com/mercadocercano/webdata-service/src/shared/httputil"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
	"github.com/mercadocercano/webdata-service/src/stats/application/usecase"
)

type StatsController struct {
	getStatsUC       *usecase.GetStatsUseCase
	getSourceStatsUC *usecase.GetSourceStatsUseCase
}

func NewStatsController(getStatsUC *usecase.GetStatsUseCase, getSourceStatsUC *usecase.GetSourceStatsUseCase) *StatsController {
	return &StatsController{getStatsUC: getStatsUC, getSourceStatsUC: getSourceStatsUC}
}

func (c *StatsController) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	stats, err := c.getStatsUC.Execute(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}

func (c *StatsController) GetSourceStats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "missing tenant")
		return
	}

	stats, err := c.getSourceStatsUC.Execute(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.JSON(w, http.StatusOK, stats)
}
