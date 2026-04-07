package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type PIMBusinessTypeProvider struct {
	baseURL string
	client  *http.Client
}

func NewPIMBusinessTypeProvider() *PIMBusinessTypeProvider {
	baseURL := os.Getenv("PIM_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://pim-service:8090"
	}
	return &PIMBusinessTypeProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type pimBusinessType struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type pimBusinessTypeResponse struct {
	Data  []pimBusinessType `json:"data"`
	Items []pimBusinessType `json:"items"`
}

func (p *PIMBusinessTypeProvider) FetchAll(ctx context.Context, tenantID string) ([]port.BusinessType, error) {
	url := fmt.Sprintf("%s/api/v1/business-types", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d", resp.StatusCode)
	}

	var body pimBusinessTypeResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// PIM may return business types in "data" or "items" depending on version
	btList := body.Data
	if len(btList) == 0 {
		btList = body.Items
	}

	result := make([]port.BusinessType, len(btList))
	for i, bt := range btList {
		result[i] = port.BusinessType{Code: bt.Code, Name: bt.Name}
	}
	return result, nil
}
