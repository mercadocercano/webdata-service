package pim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

type PIMClientAdapter struct {
	baseURL    string
	httpClient *http.Client
}

func NewPIMClientAdapter() *PIMClientAdapter {
	baseURL := os.Getenv("PIM_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://pim-service:8090"
	}
	return &PIMClientAdapter{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type enrichmentQueueResponse struct {
	Products   []port.NeedsEnrichmentItem `json:"products"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
}

func (a *PIMClientAdapter) ListNeedingEnrichment(ctx context.Context, businessType string, limit int) (*port.NeedsEnrichmentResult, error) {
	return a.listNeedingEnrichmentPage(ctx, businessType, limit, 0)
}

func (a *PIMClientAdapter) ListNeedingEnrichmentPage(ctx context.Context, businessType string, limit, offset int) (*port.NeedsEnrichmentResult, error) {
	return a.listNeedingEnrichmentPage(ctx, businessType, limit, offset)
}

func (a *PIMClientAdapter) listNeedingEnrichmentPage(ctx context.Context, businessType string, limit, offset int) (*port.NeedsEnrichmentResult, error) {
	params := url.Values{}
	if businessType != "" {
		params.Set("business_type", businessType)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/enrichment-queue", a.baseURL)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("pim enrichment queue request: %w", err)
	}
	a.setServiceHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d on enrichment-queue", resp.StatusCode)
	}

	var body enrichmentQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding enrichment queue response: %w", err)
	}

	return &port.NeedsEnrichmentResult{
		Items: body.Products,
		Total: body.Pagination.Total,
	}, nil
}

// byIDsResponse mapea la respuesta de /global-catalog/by-ids
type byIDsResponse struct {
	Products []port.NeedsEnrichmentItem `json:"products"`
	Total    int                        `json:"total"`
}

// GetProductsByIDs obtiene productos específicos del catálogo global por sus IDs.
func (a *PIMClientAdapter) GetProductsByIDs(ctx context.Context, ids []string) (*port.NeedsEnrichmentResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/by-ids?ids=%s", a.baseURL, strings.Join(ids, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("pim by-ids request: %w", err)
	}
	a.setServiceHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable on by-ids: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d on by-ids", resp.StatusCode)
	}

	var body byIDsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding by-ids response: %w", err)
	}

	return &port.NeedsEnrichmentResult{
		Items: body.Products,
		Total: body.Total,
	}, nil
}

func (a *PIMClientAdapter) UpdateProduct(ctx context.Context, productID string, updateReq port.UpdateProductRequest) error {
	payload, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshaling update request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products/%s", a.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("pim update request: %w", err)
	}
	a.setServiceHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pim-service unreachable on update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("pim-service returned status %d on update", resp.StatusCode)
	}
	return nil
}

func (a *PIMClientAdapter) setServiceHeaders(req *http.Request) {
	req.Header.Set("X-User-Role", "marketplace_admin")
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000000")
}
