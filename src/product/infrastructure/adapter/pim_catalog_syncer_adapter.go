package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/platformauth"
)

// PIMCatalogSyncerAdapter habla con catalog-service (PLAT-E39 T9b). Antes apuntaba a
// pim-service con el bypass forjable `X-User-Role: marketplace_admin` (uno de los seis
// agujeros de aislamiento que motivaron esta épica). Ahora:
//   - Lee CATALOG_SERVICE_URL (default http://mc-catalog-service:8160), no PIM_SERVICE_URL.
//   - Autentica con un JWT de servicio `system_admin` (decisión D8), verificado DENTRO de
//     catalog-service (D3.1) — nunca con un header de rol forjable por el cliente.
//   - No envía `X-Tenant-ID`: catalog-service no tiene noción de tenant (sus 14 tablas no
//     llevan tenant_id). Pasar el sentinel GlobalTenantID sería exactamente la fuga que
//     T9c prohíbe arrastrar.
//   - Update usa PATCH (T6a), no PUT. RefreshTemplateProducts y la cola apuntan a los paths
//     de catalog-service.
type PIMCatalogSyncerAdapter struct {
	baseURL string
	client  *http.Client
	tokens  *platformauth.ServiceTokenProvider
}

func NewPIMCatalogSyncerAdapter(tokens *platformauth.ServiceTokenProvider) *PIMCatalogSyncerAdapter {
	baseURL := os.Getenv("CATALOG_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://mc-catalog-service:8160"
	}
	return &PIMCatalogSyncerAdapter{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
		tokens:  tokens,
	}
}

type pimGlobalProductResponse struct {
	Data *port.PIMGlobalProduct `json:"data"`
}

type pimGlobalProductListResponse struct {
	Items []port.PIMGlobalProduct `json:"items"`
	Data  []port.PIMGlobalProduct `json:"data"`
}

// enrichmentQueueListResponse es el shape de catalog-service
// `GET /global-catalog/enrichment-queue` (T9a): `products` + `pagination.total` (paridad 1:1
// con pim). El adapter del lado product sólo consume id/name/brand/category/business_type/
// quality_score; los campos extra del summary se ignoran al deserializar.
type enrichmentQueueListResponse struct {
	Products   []port.NeedsEnrichmentItem `json:"products"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
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
	if err := a.setHeaders(req); err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
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
	params.Set("search", name)
	if brand != "" {
		params.Set("brand", brand)
	}
	params.Set("limit", "10")

	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products?%s", a.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if err := a.setHeaders(req); err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
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
	// catalog-service `search` es substring (ILIKE), igual que pim: puede devolver productos
	// parecidos pero distintos. Sólo tratamos un resultado como match existente cuando el
	// nombre es exacto (case-insensitive) — si no, devolvemos nil para que el sync cree el
	// producto en vez de pisar uno real distinto. EAN dedup maneja el resto.
	target := strings.TrimSpace(name)
	for i := range items {
		if strings.EqualFold(strings.TrimSpace(items[i].Name), target) {
			return &items[i], nil
		}
	}
	return nil, nil
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
	if err := a.setHeaders(req); err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// T6a exige X-Request-Id (correlation_id de la auditoría A09) para toda mutación
	// auditada. El enrichment batch es async/scheduled, sin inbound HTTP significativo,
	// así que se genera un UUID por write: cada mutación queda trazable en A09.
	req.Header.Set("X-Request-Id", uuid.NewString())

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog-service returned status %d on create: %s", resp.StatusCode, string(respBodyBytes))
	}

	// catalog-service (T6a) devuelve el producto creado como objeto JSON plano (sin
	// wrapper `data`), a diferencia del viejo pim. Decodificamos directo en la entidad.
	var product port.PIMGlobalProduct
	if err := json.Unmarshal(respBodyBytes, &product); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &product, nil
}

func (a *PIMCatalogSyncerAdapter) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, updateReq port.UpdatePIMProductRequest) error {
	payload, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	// T6a expone PATCH (no PUT) para mutaciones admin de global_products; el writer escribe
	// dato+auditoría A09+fila outbox D5 en una transacción.
	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products/%s", a.baseURL, id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if err := a.setHeaders(req); err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// T6a exige X-Request-Id (correlation_id de la auditoría A09) para mutaciones
	// auditadas. UUID por write para trazabilidad del batch async de enrichment.
	req.Header.Set("X-Request-Id", uuid.NewString())

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("catalog-service returned status %d on update", resp.StatusCode)
	}
	return nil
}

// RefreshTemplateProducts dispara el job de catalog-service que recalcula
// business_type_product_templates desde global_products verificados (T6d1). Se llama tras
// un batch sync.
func (a *PIMCatalogSyncerAdapter) RefreshTemplateProducts(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/refresh-template-products", a.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating refresh request: %w", err)
	}
	if err := a.setHeaders(req); err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("catalog-service unreachable for refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("catalog-service returned status %d on refresh", resp.StatusCode)
	}
	return nil
}

// ListNeedingEnrichment consulta la cola de enrichment de catalog-service
// (`GET /global-catalog/enrichment-queue`, T9a) por global_products con datos incompletos.
// Predicado "incompleto" server-side always-on (A03): el caller no inyecta SQL.
func (a *PIMCatalogSyncerAdapter) ListNeedingEnrichment(ctx context.Context, businessType string, limit, offset int) (*port.ListNeedingEnrichmentResult, error) {
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
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if err := a.setHeaders(req); err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog-service returned status %d on enrichment-queue", resp.StatusCode)
	}

	var body enrichmentQueueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &port.ListNeedingEnrichmentResult{
		Items: body.Products,
		Total: body.Pagination.Total,
	}, nil
}

// setHeaders pone SOLO el Authorization con el JWT de servicio (D8). Sin X-User-Role
// (forjable, D3.1) ni X-Tenant-ID (catalog-service no tiene tenant).
func (a *PIMCatalogSyncerAdapter) setHeaders(req *http.Request) error {
	token, err := a.tokens.Token()
	if err != nil {
		return fmt.Errorf("platform service token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
