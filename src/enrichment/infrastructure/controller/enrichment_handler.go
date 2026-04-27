package controller

import (
	"encoding/json"
	"net/http"

	"github.com/mercadocercano/webdata-service/src/enrichment/application"
	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type EnrichmentHandler struct {
	runBatch  *application.RunEnrichmentBatchUseCase
	getStatus *application.GetEnrichmentStatusUseCase
}

func NewEnrichmentHandler(
	runBatch *application.RunEnrichmentBatchUseCase,
	getStatus *application.GetEnrichmentStatusUseCase,
) *EnrichmentHandler {
	return &EnrichmentHandler{runBatch: runBatch, getStatus: getStatus}
}

// RunBatch POST /api/v1/enrichment/run
func (h *EnrichmentHandler) RunBatch(w http.ResponseWriter, r *http.Request) {
	var req application.RunEnrichmentBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.runBatch.Execute(r.Context(), req)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, result)
}

// GetStatus GET /api/v1/enrichment/status
func (h *EnrichmentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.getStatus.Execute(r.Context())
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.JSONResponse(w, http.StatusOK, status)
}
