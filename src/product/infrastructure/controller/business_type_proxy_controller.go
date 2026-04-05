package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mercadocercano/webdata-service/src/shared/middleware"
)

type BusinessTypeProxyController struct {
	pimBaseURL string
	client     *http.Client
}

func NewBusinessTypeProxyController() *BusinessTypeProxyController {
	baseURL := os.Getenv("PIM_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://pim-service:8090"
	}
	return &BusinessTypeProxyController{
		pimBaseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GET /api/v1/business-types — proxies to pim-service
func (c *BusinessTypeProxyController) ListBusinessTypes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		middleware.JSONError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	url := fmt.Sprintf("%s/api/v1/business-types", c.pimBaseURL)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		middleware.JSONError(w, http.StatusInternalServerError, "failed to create proxy request")
		return
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		middleware.JSONError(w, http.StatusBadGateway, fmt.Sprintf("pim-service unreachable: %v", err))
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
