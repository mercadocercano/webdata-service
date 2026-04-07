package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

type PIMCatalogSyncerAdapter struct {
	baseURL string
	client  *http.Client
}

func NewPIMCatalogSyncerAdapter() *PIMCatalogSyncerAdapter {
	baseURL := os.Getenv("PIM_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://pim-service:8090"
	}
	return &PIMCatalogSyncerAdapter{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type pimGlobalProductResponse struct {
	Data *port.PIMGlobalProduct `json:"data"`
}

type pimGlobalProductListResponse struct {
	Items []port.PIMGlobalProduct `json:"items"`
	Data  []port.PIMGlobalProduct `json:"data"`
}

func (a *PIMCatalogSyncerAdapter) SearchByEAN(ctx context.Context, tenantID uuid.UUID, ean string) (*port.PIMGlobalProduct, error) {
	if ean == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products?ean=%s&limit=1", a.baseURL, url.QueryEscape(ean))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(req, tenantID)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var body pimGlobalProductListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	items := body.Items
	if len(items) == 0 {
		items = body.Data
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (a *PIMCatalogSyncerAdapter) SearchByNameBrand(ctx context.Context, tenantID uuid.UUID, name, brand string) (*port.PIMGlobalProduct, error) {
	params := url.Values{}
	params.Set("search_name", name)
	if brand != "" {
		params.Set("search_brand", brand)
	}
	params.Set("limit", "1")

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products?%s", a.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(req, tenantID)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var body pimGlobalProductListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	items := body.Items
	if len(items) == 0 {
		items = body.Data
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (a *PIMCatalogSyncerAdapter) Create(ctx context.Context, tenantID uuid.UUID, createReq port.CreatePIMProductRequest) (*port.PIMGlobalProduct, error) {
	payload, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products", a.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(req, tenantID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pim-service returned status %d on create", resp.StatusCode)
	}

	var body pimGlobalProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return body.Data, nil
}

func (a *PIMCatalogSyncerAdapter) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, updateReq port.UpdatePIMProductRequest) error {
	payload, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products/%s", a.baseURL, id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(req, tenantID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("pim-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("pim-service returned status %d on update", resp.StatusCode)
	}
	return nil
}

func (a *PIMCatalogSyncerAdapter) setHeaders(req *http.Request, tenantID uuid.UUID) {
	req.Header.Set("X-User-Role", "marketplace_admin")
	if token := authTokenFromContext(req.Context()); token != "" {
		req.Header.Set("Authorization", token)
		if jwtTenantID := tenantIDFromContext(req.Context()); jwtTenantID != "" {
			req.Header.Set("X-Tenant-ID", jwtTenantID)
		} else {
			req.Header.Set("X-Tenant-ID", tenantID.String())
		}
	} else {
		req.Header.Set("X-Tenant-ID", tenantID.String())
	}
}

type contextKey string

const authTokenKey contextKey = "auth_token"
const jwtTenantKey contextKey = "jwt_tenant_id"

func ContextWithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey, token)
}

func ContextWithJWTTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, jwtTenantKey, tenantID)
}

func authTokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authTokenKey).(string); ok {
		return v
	}
	return ""
}

func tenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(jwtTenantKey).(string); ok {
		return v
	}
	return ""
}
