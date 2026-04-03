package controller

import (
	"net/http"

	statsusecase "github.com/mercadocercano/webdata-service/src/stats/application/usecase"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type StatsController struct {
	statsUC       *statsusecase.GetStatsUseCase
	sourceStatsUC *statsusecase.GetSourceStatsUseCase
}

func NewStatsController(statsUC *statsusecase.GetStatsUseCase, sourceStatsUC *statsusecase.GetSourceStatsUseCase) *StatsController {
	return &StatsController{statsUC: statsUC, sourceStatsUC: sourceStatsUC}
}

func (c *StatsController) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	resp, err := c.statsUC.Execute(r.Context(), tenantID)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}

func (c *StatsController) GetSourceStats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	resp, err := c.sourceStatsUC.Execute(r.Context(), tenantID)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": resp})
}
