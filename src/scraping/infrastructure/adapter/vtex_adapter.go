package adapter

// vtex_adapter.go — HTTP-JSON adapter for VTEX storefronts.
//
// Design decision: this adapter is VTEX-specific (Cordiez, and future VTEX stores)
// rather than a generic JSON-mapping adapter. A declarative json_mapping approach
// was considered but rejected for this scope:
//   - VTEX's nested structure (items[0].sellers[0].commertialOffer.Price) is hard
//     to express cleanly with flat jsonpath rules.
//   - A typed Go struct is safer, faster, and easier to test.
//   - The port (FetchHTTPJSON) is already generic; if a non-VTEX store needs
//     http_json, a second adapter can be wired in the use case switch.
//
// Pagination: VTEX uses range-based pagination via _from/_to query params and
// returns HTTP 206 with a Content-Range: items 0-49/TOTAL header.
// BuildPaginatedURL (page-number style) is NOT used here.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

const (
	vtexDefaultPageSize = 50
	vtexMaxPageSize     = 50  // VTEX hard limit per request
	vtexHTTPTimeout     = 30 * time.Second
	vtexMaxOffset       = 2500 // VTEX public API rejects _from > 2500 with HTTP 400.
	// For categories with >2500 products (e.g. Carrefour Almacén=6101), seed
	// subcategories separately to get full coverage.
)

// VTEXAdapter fetches product catalogs from VTEX storefronts via their
// public catalog_system/pub/products/search API (no auth required).
type VTEXAdapter struct {
	httpClient *http.Client
}

// NewVTEXAdapter creates a production-ready VTEX adapter.
func NewVTEXAdapter() *VTEXAdapter {
	return &VTEXAdapter{
		httpClient: &http.Client{
			Timeout: vtexHTTPTimeout,
		},
	}
}

// newVTEXAdapterWithClient creates a VTEX adapter with a custom HTTP client.
// Intended for use in tests only.
func newVTEXAdapterWithClient(client *http.Client) *VTEXAdapter {
	return &VTEXAdapter{httpClient: client}
}

// vtexProduct represents a single product in the VTEX catalog search response.
// Only the fields we map to RawProduct are included; the rest are ignored.
type vtexProduct struct {
	ProductName string   `json:"productName"`
	Brand       string   `json:"brand"`
	Link        string   `json:"link"`
	LinkText    string   `json:"linkText"`
	Categories  []string `json:"categories"`
	Items       []vtexItem
}

type vtexItem struct {
	ItemID   string       `json:"itemId"`
	EAN      string       `json:"ean"`
	Images   []vtexImage  `json:"images"`
	Sellers  []vtexSeller `json:"sellers"`
}

type vtexImage struct {
	ImageURL string `json:"imageUrl"`
}

type vtexSeller struct {
	CommertialOffer vtexOffer `json:"commertialOffer"`
}

// vtexOffer holds pricing and availability from VTEX's commertialOffer.
// Note: VTEX spells "commertialOffer" with one 'm' — kept as-is to match API.
type vtexOffer struct {
	Price       float64 `json:"Price"`
	ListPrice   float64 `json:"ListPrice"`
	IsAvailable bool    `json:"IsAvailable"`
}

// FetchHTTPJSON fetches all products from the VTEX search API, paginating
// with range params (_from/_to) until the full catalog is retrieved or
// opts.MaxProducts is reached. HTTP 206 is treated as success.
func (a *VTEXAdapter) FetchHTTPJSON(ctx context.Context, baseURL string, opts scrapingport.HTTPJSONOptions) ([]scrapingport.RawProduct, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 || pageSize > vtexMaxPageSize {
		pageSize = vtexDefaultPageSize
	}

	var (
		all   []scrapingport.RawProduct
		from  = 0
		total = -1 // unknown until first response
	)

	for {
		// Check context cancellation before each page fetch.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Stop if we already know the total and have fetched everything.
		if total >= 0 && from >= total {
			break
		}

		// VTEX public API rejects _from > 2500 with HTTP 400.
		// Stop gracefully at the cap — for categories with more products,
		// seed subcategories separately to get full coverage.
		if from >= vtexMaxOffset {
			break
		}

		// Apply MaxProducts cap: don't request more than what's left.
		remaining := pageSize
		if opts.MaxProducts > 0 {
			left := opts.MaxProducts - len(all)
			if left <= 0 {
				break
			}
			if left < remaining {
				remaining = left
			}
		}

		to := from + remaining - 1

		pageURL, err := buildVTEXPageURL(baseURL, from, to)
		if err != nil {
			return nil, fmt.Errorf("vtex: building page URL: %w", err)
		}

		products, pageTotal, err := a.fetchPage(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("vtex: fetching page _from=%d: %w", from, err)
		}

		// On the first request, learn the total from Content-Range.
		if total < 0 {
			total = pageTotal
		}

		all = append(all, products...)

		if len(products) == 0 {
			// Safety: no products returned — avoid infinite loop.
			break
		}

		from += len(products)
	}

	return all, nil
}

// fetchPage performs a single HTTP GET to the VTEX API and parses the response.
// Returns the products on that page and the total product count (from Content-Range).
func (a *VTEXAdapter) fetchPage(ctx context.Context, pageURL string) ([]scrapingport.RawProduct, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mercado-cercano-webdata/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// VTEX returns HTTP 206 Partial Content for paginated results — treat as success.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse total from Content-Range (older VTEX: "items 0-49/409") or
	// Resources (newer VTEX storefronts: "0-49/409"). Try both headers.
	total := parseContentRangeTotal(resp.Header.Get("Content-Range"))
	if total < 0 {
		total = parseContentRangeTotal(resp.Header.Get("Resources"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response body: %w", err)
	}

	var vtexProducts []vtexProduct
	if err := json.Unmarshal(body, &vtexProducts); err != nil {
		return nil, 0, fmt.Errorf("parsing VTEX JSON response: %w", err)
	}

	products := make([]scrapingport.RawProduct, 0, len(vtexProducts))
	for _, vp := range vtexProducts {
		products = append(products, mapVTEXProduct(vp, baseURLFromPageURL(pageURL)))
	}

	return products, total, nil
}

// mapVTEXProduct converts a VTEX product to the canonical RawProduct struct.
func mapVTEXProduct(vp vtexProduct, storeBaseURL string) scrapingport.RawProduct {
	rp := scrapingport.RawProduct{
		Title:    vp.ProductName,
		Brand:    vp.Brand,
		Category: firstCategory(vp.Categories),
		URL:      productURL(vp, storeBaseURL),
	}

	if len(vp.Items) > 0 {
		item := vp.Items[0]
		rp.SKU = item.ItemID
		rp.EAN = item.EAN

		if len(item.Images) > 0 {
			rp.ImageURL = item.Images[0].ImageURL
		}

		if len(item.Sellers) > 0 {
			offer := item.Sellers[0].CommertialOffer
			price := offer.Price
			listPrice := offer.ListPrice
			inStock := offer.IsAvailable

			rp.Price = &price
			rp.OriginalPrice = &listPrice
			rp.InStock = &inStock
		}
	}

	return rp
}

// productURL resolves the product URL from the VTEX response.
// Prefers the direct link field; falls back to constructing from linkText + store base URL.
func productURL(vp vtexProduct, storeBaseURL string) string {
	if vp.Link != "" {
		return vp.Link
	}
	if vp.LinkText != "" && storeBaseURL != "" {
		// VTEX linkText is the URL slug: combine with the store root.
		u, err := url.Parse(storeBaseURL)
		if err == nil {
			u.Path = "/" + vp.LinkText + "/p"
			u.RawQuery = ""
			return u.String()
		}
	}
	return ""
}

// firstCategory returns the first entry from the VTEX categories array.
// VTEX categories are path strings like "/Lácteos/Leches/...".
func firstCategory(categories []string) string {
	if len(categories) > 0 {
		return categories[0]
	}
	return ""
}

// buildVTEXPageURL appends _from and _to range params to the base URL.
// Preserves any existing query params (e.g. fq=C:/10003/).
func buildVTEXPageURL(baseURL string, from, to int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", baseURL, err)
	}
	q := u.Query()
	q.Set("_from", strconv.Itoa(from))
	q.Set("_to", strconv.Itoa(to))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// parseContentRangeTotal extracts the total count from a Content-Range or
// Resources header. VTEX storefronts use different headers depending on the
// platform version:
//
//   - Cordiez / older VTEX: "Content-Range: items 0-49/409"
//   - Día / Carrefour / La Anónima / Farmacity: "resources: 0-49/409"
//
// Both follow the same "range/total" format. Returns -1 on parse failure.
func parseContentRangeTotal(header string) int {
	if header == "" {
		return -1
	}
	// Strip optional "items " prefix (VTEX uses this format in Content-Range).
	parts := strings.Split(header, "/")
	if len(parts) != 2 {
		return -1
	}
	total, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return -1
	}
	return total
}

// baseURLFromPageURL strips _from/_to params to recover the store root URL
// for use in URL construction (productURL fallback).
func baseURLFromPageURL(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return pageURL
	}
	q := u.Query()
	q.Del("_from")
	q.Del("_to")
	u.RawQuery = q.Encode()
	// Return scheme + host only for slug construction.
	u.Path = ""
	u.RawQuery = ""
	return u.Scheme + "://" + u.Host
}
