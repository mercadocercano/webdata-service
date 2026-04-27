package mercadolibre

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type MLAdapter struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	tokenURL     string
	searchURL    string

	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

func NewMLAdapter() *MLAdapter {
	return NewMLAdapterWithURLs(
		"https://api.mercadolibre.com/oauth/token",
		"https://api.mercadolibre.com/products/search",
		os.Getenv("ML_CLIENT_ID"),
		os.Getenv("ML_CLIENT_SECRET"),
	)
}

// NewMLAdapterWithURLs permite inyectar URLs custom para testing.
func NewMLAdapterWithURLs(tokenURL, searchURL, clientID, clientSecret string) *MLAdapter {
	return &MLAdapter{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		tokenURL:     tokenURL,
		searchURL:    searchURL,
	}
}

func (a *MLAdapter) SourceName() entity.Source {
	return entity.SourceML
}

func (a *MLAdapter) FindByGTIN(ctx context.Context, gtin string) (*entity.ProductData, error) {
	if a.clientID == "" || a.clientSecret == "" {
		return nil, nil
	}
	return a.search(ctx, gtin)
}

func (a *MLAdapter) FindByName(ctx context.Context, name string) (*entity.ProductData, error) {
	if a.clientID == "" || a.clientSecret == "" {
		return nil, nil
	}
	return a.search(ctx, name)
}

func (a *MLAdapter) search(ctx context.Context, query string) (*entity.ProductData, error) {
	token, err := a.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("ml get token: %w", err)
	}

	endpoint := fmt.Sprintf(
		"%s?site_id=MLA&q=%s&status=active",
		a.searchURL,
		url.QueryEscape(query),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("ml creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ml request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return parseMLSearchResponse(resp.Body)
}

func (a *MLAdapter) getToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	return a.refreshToken(ctx)
}

func (a *MLAdapter) refreshToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.accessToken, nil
	}

	body := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {a.clientID},
		"client_secret": {a.clientSecret},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.tokenURL,
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	a.accessToken = tok.AccessToken
	// Refresh 10 min before expiry
	a.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn-600) * time.Second)
	return a.accessToken, nil
}

type mlSearchResponse struct {
	Results []mlProduct `json:"results"`
}

type mlProduct struct {
	Name       string        `json:"name"`
	DomainID   string        `json:"domain_id"`
	Pictures   []mlPicture   `json:"pictures"`
	Attributes []mlAttribute `json:"attributes"`
}

type mlPicture struct {
	URL string `json:"url"`
}

type mlAttribute struct {
	ID     string           `json:"id"`
	Values []mlAttrValue    `json:"values"`
}

type mlAttrValue struct {
	Name string `json:"name"`
}

func parseMLSearchResponse(body io.Reader) (*entity.ProductData, error) {
	var response mlSearchResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding ml response: %w", err)
	}

	if len(response.Results) == 0 {
		return nil, nil
	}

	product := response.Results[0]
	data := &entity.ProductData{
		Name:     product.Name,
		Category: strings.TrimPrefix(product.DomainID, "MLA-"),
		Source:   entity.SourceML,
	}

	if len(product.Pictures) > 0 {
		data.ImageURL = product.Pictures[0].URL
	}

	for _, attr := range product.Attributes {
		if len(attr.Values) == 0 {
			continue
		}
		switch attr.ID {
		case "BRAND":
			data.Brand = attr.Values[0].Name
		case "EAN":
			data.EAN = attr.Values[0].Name
		case "GTIN":
			data.GTIN = attr.Values[0].Name
		}
	}

	return data, nil
}
