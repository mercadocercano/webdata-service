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
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/platformauth"
)

// PIMClientAdapter habla con catalog-service (PLAT-E39 T9b). Antes apuntaba a pim-service
// con el bypass forjable `X-User-Role: marketplace_admin`. Ahora autentica con el JWT de
// servicio `system_admin` (D8), lee CATALOG_SERVICE_URL y usa los paths/verbos de
// catalog-service (PATCH, no PUT).
type PIMClientAdapter struct {
	baseURL    string
	httpClient *http.Client
	tokens     *platformauth.ServiceTokenProvider
}

func NewPIMClientAdapter(tokens *platformauth.ServiceTokenProvider) *PIMClientAdapter {
	baseURL := os.Getenv("CATALOG_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://mc-catalog-service:8160"
	}
	return &PIMClientAdapter{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		tokens:     tokens,
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
		return nil, fmt.Errorf("catalog enrichment queue request: %w", err)
	}
	if err := a.setServiceHeaders(req); err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog-service returned status %d on enrichment-queue", resp.StatusCode)
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

// catalogProductDetail es el subset de `GlobalProductReadResponse` (T6d1) que el enrichment
// consume: catalog-service `GET /global-catalog/products/:id` responde `{"product": {...}}`
// con el producto como objeto plano.
type catalogProductDetail struct {
	ID           string  `json:"id"`
	EAN          *string `json:"ean"`
	Name         string  `json:"name"`
	Brand        *string `json:"brand"`
	Category     *string `json:"category"`
	BusinessType *string `json:"business_type"`
	QualityScore int     `json:"quality_score"`
}
type catalogProductDetailResponse struct {
	Product catalogProductDetail `json:"product"`
}

// GetProductsByIDs obtiene productos específicos del catálogo global por sus IDs.
//
// catalog-service NO expone `by-ids` (T9a dejó ese batch para confirmar en T9b). Agregar un
// endpoint `by-ids` sería superficie nueva sobre un servicio L4 y, por el precedente de esta
// épica (T8→T8a/b/c, T6d→T6d1/d2, T9→T9a/b), merece sub-tarea con gate propio — no se
// bundlea acá. En su lugar se hace fan-out sobre `GET /global-catalog/products/:id` (T6d1,
// ya gateado), que reusa superficie existente. El on-demand está acotado a ≤100 IDs
// (validado en el usecase), así que el costo de N requests es aceptable y no abre superficie.
func (a *PIMClientAdapter) GetProductsByIDs(ctx context.Context, ids []string) (*port.NeedsEnrichmentResult, error) {
	token, err := a.tokens.Token()
	if err != nil {
		return nil, fmt.Errorf("platform service token: %w", err)
	}

	items := make([]port.NeedsEnrichmentItem, 0, len(ids))
	for _, id := range ids {
		endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products/%s", a.baseURL, url.PathEscape(id))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("catalog by-id request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("catalog-service unreachable on by-id %s: %w", id, err)
		}
		if resp.StatusCode == http.StatusNotFound {
			// Producto que ya no existe: se saltea, no aborta el lote.
			resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("catalog-service returned status %d on by-id %s", resp.StatusCode, id)
		}

		var body catalogProductDetailResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding by-id response for %s: %w", id, err)
		}
		resp.Body.Close()

		items = append(items, mapDetailToNeedsEnrichmentItem(body.Product))
	}

	return &port.NeedsEnrichmentResult{
		Items: items,
		Total: len(items),
	}, nil
}

func mapDetailToNeedsEnrichmentItem(d catalogProductDetail) port.NeedsEnrichmentItem {
	ean, brand, category, businessType := "", "", "", ""
	if d.EAN != nil {
		ean = *d.EAN
	}
	if d.Brand != nil {
		brand = *d.Brand
	}
	if d.Category != nil {
		category = *d.Category
	}
	if d.BusinessType != nil {
		businessType = *d.BusinessType
	}
	return port.NeedsEnrichmentItem{
		ID:           d.ID,
		EAN:          ean,
		Name:         d.Name,
		Brand:        brand,
		Category:     category,
		BusinessType: businessType,
		QualityScore: d.QualityScore,
	}
}

func (a *PIMClientAdapter) UpdateProduct(ctx context.Context, productID string, updateReq port.UpdateProductRequest) error {
	payload, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshaling update request: %w", err)
	}

	// T6a expone PATCH (no PUT) para mutaciones admin; el writer escribe dato+auditoría A09
	// +fila outbox D5 en una transacción.
	endpoint := fmt.Sprintf("%s/api/v1/global-catalog/products/%s", a.baseURL, url.PathEscape(productID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("catalog update request: %w", err)
	}
	if err := a.setServiceHeaders(req); err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// T6a exige X-Request-Id (correlation_id de la auditoría A09) para mutaciones
	// auditadas. UUID por write para trazabilidad del batch async de enrichment.
	req.Header.Set("X-Request-Id", uuid.NewString())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("catalog-service unreachable on update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("catalog-service returned status %d on update", resp.StatusCode)
	}
	return nil
}

// setServiceHeaders pone SOLO el Authorization con el JWT de servicio (D8). Sin X-User-Role
// (forjable, D3.1) ni X-Tenant-ID (catalog-service no tiene tenant).
func (a *PIMClientAdapter) setServiceHeaders(req *http.Request) error {
	token, err := a.tokens.Token()
	if err != nil {
		return fmt.Errorf("platform service token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
