package openfoodfacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

const userAgent = "MercadoCercano/1.0 (contacto@mercadocercano.com)"

type OFFAdapter struct {
	httpClient *http.Client
	baseURL    string
}

func NewOFFAdapter() *OFFAdapter {
	return NewOFFAdapterWithBaseURL("https://world.openfoodfacts.org")
}

// NewOFFAdapterWithBaseURL permite inyectar URL base para testing.
func NewOFFAdapterWithBaseURL(baseURL string) *OFFAdapter {
	return &OFFAdapter{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
	}
}

func (a *OFFAdapter) SourceName() entity.Source {
	return entity.SourceOFF
}

func (a *OFFAdapter) FindByGTIN(ctx context.Context, gtin string) (*entity.ProductData, error) {
	if gtin == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf(
		"%s/api/v2/product/%s.json?fields=product_name,brands,categories_tags,image_front_url,image_url,code",
		a.baseURL,
		url.PathEscape(gtin),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("off creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("off request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseOFFProductResponse(resp.Body)
}

func (a *OFFAdapter) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	if name == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf(
		"%s/cgi/search.pl?search_terms=%s&json=1&page_size=1&fields=product_name,brands,categories_tags,image_front_url,image_url,code",
		a.baseURL,
		url.QueryEscape(name),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("off creating search request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("off search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseOFFSearchResponse(resp.Body)
}

type offProductResponse struct {
	Status  int        `json:"status"`
	Product offProduct `json:"product"`
}

type offSearchResponse struct {
	Count    int          `json:"count"`
	Products []offProduct `json:"products"`
}

type offProduct struct {
	ProductName     string   `json:"product_name"`
	Brands          string   `json:"brands"`
	CategoriesTags  []string `json:"categories_tags"`
	ImageFrontURL   string   `json:"image_front_url"`
	ImageURL        string   `json:"image_url"`
	Code            string   `json:"code"`
}

func parseOFFProductResponse(body io.Reader) (*entity.ProductData, error) {
	var response offProductResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding off product response: %w", err)
	}

	if response.Status != 1 {
		return nil, nil
	}

	return buildProductData(response.Product), nil
}

func parseOFFSearchResponse(body io.Reader) (*entity.ProductData, error) {
	var response offSearchResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding off search response: %w", err)
	}

	if len(response.Products) == 0 {
		return nil, nil
	}

	return buildProductData(response.Products[0]), nil
}

func buildProductData(p offProduct) *entity.ProductData {
	imageURL := p.ImageFrontURL
	if imageURL == "" {
		imageURL = p.ImageURL
	}

	category := ""
	if len(p.CategoriesTags) > 0 {
		category = strings.TrimPrefix(p.CategoriesTags[0], "en:")
	}

	return &entity.ProductData{
		EAN:      p.Code,
		Name:     p.ProductName,
		Brand:    p.Brands,
		Category: category,
		ImageURL: imageURL,
		Source:   entity.SourceOFF,
	}
}
