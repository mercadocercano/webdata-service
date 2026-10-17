package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/platformauth"
)

// PIMBusinessTypeProvider lee los business types de catalog-service (módulo `businesstype`
// de T5). Antes apuntaba a pim con `X-User-Role: marketplace_admin`; ahora autentica con el
// JWT de servicio D8 y lee el shape de catalog-service (`business_types` + `pagination`).
type PIMBusinessTypeProvider struct {
	baseURL string
	client  *http.Client
	tokens  *platformauth.ServiceTokenProvider
}

func NewPIMBusinessTypeProvider(tokens *platformauth.ServiceTokenProvider) *PIMBusinessTypeProvider {
	baseURL := os.Getenv("CATALOG_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://mc-catalog-service:8160"
	}
	return &PIMBusinessTypeProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		tokens:  tokens,
	}
}

type pimBusinessType struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// pimBusinessTypeResponse matchea el shape de catalog-service
// `GET /api/v1/business-types` (T5 + T20): `{"items":[...], "total_pages":N}`.
// T20 alineó el envelope admin de catalog-service a `cr.ListResponse` de go-shared
// (`{items, total_count, page, page_size, total_pages}`) — decisión del owner 2026-08-06
// ("catalog-service a items"). El viejo pim devolvía `data`/`items`; catalog-service
// adoptó `business_types` en T5 y T20 lo revierte a `items`.
type pimBusinessTypeResponse struct {
	Items      []pimBusinessType `json:"items"`
	TotalPages int               `json:"total_pages"`
}

func (p *PIMBusinessTypeProvider) FetchAll(ctx context.Context, tenantID string) ([]port.BusinessType, error) {
	token, err := p.tokens.Token()
	if err != nil {
		return nil, fmt.Errorf("platform service token: %w", err)
	}

	var allTypes []port.BusinessType
	page := 1

	for {
		url := fmt.Sprintf("%s/api/v1/business-types?page=%d&page_size=100", p.baseURL, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		// D8: JWT de servicio. Sin X-User-Role (forjable) ni X-Tenant-ID (catalog no tiene tenant).
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("catalog-service unreachable: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("catalog-service returned status %d", resp.StatusCode)
		}

		var body pimBusinessTypeResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(body.Items) == 0 {
			break
		}

		for _, bt := range body.Items {
			allTypes = append(allTypes, port.BusinessType{Code: bt.Code, Name: bt.Name})
		}

		// Si trajimos menos de page_size, ya tenemos todo.
		if len(body.Items) < 100 || page >= body.TotalPages {
			break
		}
		page++
	}

	return allTypes, nil
}
