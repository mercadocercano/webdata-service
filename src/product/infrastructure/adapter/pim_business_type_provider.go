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
	Data       []pimBusinessType `json:"data"`
	Items      []pimBusinessType `json:"items"`
	TotalCount int               `json:"total_count"`
	TotalPages int               `json:"total_pages"`
}

func (p *PIMBusinessTypeProvider) FetchAll(ctx context.Context, tenantID string) ([]port.BusinessType, error) {
	var allTypes []port.BusinessType
	page := 1

	for {
		url := fmt.Sprintf("%s/api/v1/business-types?page=%d&page_size=100", p.baseURL, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-Role", "marketplace_admin")

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

		if len(btList) == 0 {
			break
		}

		for _, bt := range btList {
			allTypes = append(allTypes, port.BusinessType{Code: bt.Code, Name: bt.Name})
		}

		// If we got fewer than page_size, we've fetched everything
		if len(btList) < 100 || page >= body.TotalPages {
			break
		}
		page++
	}

	return allTypes, nil
}
