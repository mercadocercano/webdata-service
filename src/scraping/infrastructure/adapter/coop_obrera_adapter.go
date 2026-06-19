package adapter

// coop_obrera_adapter.go — HTTP-JSON adapter for Cooperativa Obrera (La Coope en Casa).
//
// The API is a private Angular SPA internal API, public and unauthenticated.
// Endpoint: POST https://api.lacoopeencasa.coop/api/articulos/pagina
//
// Key gotchas (confirmed 2026-06-17):
//   - "tipo_seleccion" MUST be nested inside "filtros", NOT at root level.
//     Root-level → estado:4 "El campo tipo_seleccion es requerido".
//   - "filtros.categoria" must be empty [] to traverse the full root category.
//   - "id_busqueda" must be a ROOT level-1 category id (2-7). Leaf ids return 0 products.
//   - Page size is fixed at 8 server-side ("cant_articulos" is ignored).
//   - Pagination: increment "pagina" 0,1,2... until "datos.articulos" is empty.
//   - 6 root categories: Almacén=2, Frescos=3, Bebidas=4, Perfumería=5, Limpieza=6, Casa y Jardín=7.
//   - ~12k products total → ~1500 requests per full run.
//   - Image URLs are absolute (host: www.lacoopeencasa.coop/media/...).
//   - cod_interno is NOT an EAN. Dedup is by name+brand (already supported).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
)

const (
	coopDefaultRateDelay = 200 * time.Millisecond // soft rate-limit between requests
	coopHTTPTimeout      = 45 * time.Second
	coopMaxEmptyPages    = 2 // stop after N consecutive empty pages (safety net)
)

// coopRequestBody is the POST body for the Coop. Obrera paginated product endpoint.
// CRITICAL: tipo_seleccion MUST be inside filtros, not at root.
type coopRequestBody struct {
	IDBusqueda int           `json:"id_busqueda"`
	Pagina     int           `json:"pagina"`
	Filtros    coopFiltros   `json:"filtros"`
	Orden      int           `json:"orden"`
}

type coopFiltros struct {
	PrecioMenor     int           `json:"preciomenor"`
	PrecioMayor     int           `json:"preciomayor"`
	Marca           []interface{} `json:"marca"`
	Categoria       []interface{} `json:"categoria"`
	TipoSeleccion   string        `json:"tipo_seleccion"`
	FiltrosGramaje  []interface{} `json:"filtros_gramaje"`
	CantArticulos   int           `json:"cant_articulos"`
	Ofertas         bool          `json:"ofertas"`
}

// coopResponse is the API envelope: {estado, mensaje, datos, ticket}.
// estado=1 → OK. datos.articulos is the product list.
// GOTCHA: "ticket" can be a string OR an object depending on the request —
// use json.RawMessage to ignore its type safely.
type coopResponse struct {
	Estado  int              `json:"estado"`
	Mensaje string           `json:"mensaje"`
	Datos   coopDatos        `json:"datos"`
	Ticket  json.RawMessage  `json:"ticket"`
}

type coopDatos struct {
	Articulos         []coopArticulo `json:"articulos"`
	CantidadArticulos int            `json:"cantidad_articulos"`
}

// coopArticulo represents a single product from the Coop. Obrera API.
// GOTCHA: precio, precio_promo, precio_anterior come as quoted strings ("1234.00")
// OR as null — NOT as JSON numbers. Use json.Number (or *json.Number) to decode
// and convert manually. A *json.Number handles both null and "1234.00".
type coopArticulo struct {
	CodInterno     string          `json:"cod_interno"`
	Descripcion    string          `json:"descripcion"`
	MarcaDesc      string          `json:"marca_desc"`
	Precio         *json.Number    `json:"precio"`
	PrecioPromo    *json.Number    `json:"precio_promo"`
	PrecioAnterior *json.Number    `json:"precio_anterior"`
	Imagen         string          `json:"imagen"`
	IDCategoria    json.Number     `json:"id_categoria"`
}

// parseCoopPrice converts a *json.Number (which may be null or "1234.00") to float64.
func parseCoopPrice(n *json.Number) float64 {
	if n == nil {
		return 0
	}
	f, err := strconv.ParseFloat(string(*n), 64)
	if err != nil {
		return 0
	}
	return f
}

// CoopObreraAdapter fetches product catalogs from the Coop. Obrera internal JSON API.
// It uses POST requests with page_increment pagination (pagina 0,1,2... until empty).
type CoopObreraAdapter struct {
	httpClient *http.Client
	rateDelay  time.Duration
}

// NewCoopObreraAdapter creates a production-ready Coop. Obrera adapter.
func NewCoopObreraAdapter() *CoopObreraAdapter {
	return &CoopObreraAdapter{
		httpClient: &http.Client{Timeout: coopHTTPTimeout},
		rateDelay:  coopDefaultRateDelay,
	}
}

// newCoopObreraAdapterWithClient creates an adapter with a custom HTTP client.
// Intended for use in tests only.
func newCoopObreraAdapterWithClient(client *http.Client) *CoopObreraAdapter {
	return &CoopObreraAdapter{
		httpClient: client,
		rateDelay:  0, // no delay in tests
	}
}

// FetchHTTPJSON implements the page_increment pagination strategy for Coop. Obrera.
// It increments "pagina" from 0 until the response articulos list is empty.
// The baseURL should be: https://api.lacoopeencasa.coop/api/articulos/pagina
// opts.CategoryID must be set to the root category id (2-7).
func (a *CoopObreraAdapter) FetchHTTPJSON(ctx context.Context, baseURL string, opts scrapingport.HTTPJSONOptions) ([]scrapingport.RawProduct, error) {
	if opts.CategoryID == 0 {
		return nil, fmt.Errorf("coop_obrera: CategoryID is required (2=Almacén, 3=Frescos, 4=Bebidas, 5=Perfumería, 6=Limpieza, 7=Casa y Jardín)")
	}

	var (
		all         []scrapingport.RawProduct
		page        = 0
		emptyCount  = 0
	)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		articles, err := a.fetchPage(ctx, baseURL, opts.CategoryID, page)
		if err != nil {
			return nil, fmt.Errorf("coop_obrera: fetching page %d (category %d): %w", page, opts.CategoryID, err)
		}

		if len(articles) == 0 {
			emptyCount++
			if emptyCount >= coopMaxEmptyPages {
				break
			}
			page++
			continue
		}

		emptyCount = 0
		for _, a := range articles {
			all = append(all, mapCoopArticulo(a))
		}

		page++

		// Rate-limit: brief pause between requests to avoid hammering the API.
		if a.rateDelay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(a.rateDelay):
			}
		}
	}

	return all, nil
}

// fetchPage performs a single POST to the Coop. Obrera API and returns the articulos list.
func (a *CoopObreraAdapter) fetchPage(ctx context.Context, baseURL string, categoryID, page int) ([]coopArticulo, error) {
	body := coopRequestBody{
		IDBusqueda: categoryID,
		Pagina:     page,
		Filtros: coopFiltros{
			PrecioMenor:    -1,
			PrecioMayor:    -1,
			Marca:          []interface{}{},
			Categoria:      []interface{}{},
			TipoSeleccion:  "categoria",
			FiltrosGramaje: []interface{}{},
			CantArticulos:  8,
			Ofertas:        false,
		},
		Orden: 0,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mercado-cercano-webdata/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var coopResp coopResponse
	if err := json.Unmarshal(respBody, &coopResp); err != nil {
		return nil, fmt.Errorf("parsing Coop. Obrera JSON response: %w", err)
	}

	if coopResp.Estado != 1 {
		return nil, fmt.Errorf("coop_obrera API error estado=%d: %s", coopResp.Estado, coopResp.Mensaje)
	}

	return coopResp.Datos.Articulos, nil
}

// mapCoopArticulo converts a Coop. Obrera API article to the canonical RawProduct.
// Price selection: precio > 0 → precio; fallback to precio_promo; fallback to precio_anterior.
// EAN is NOT set (cod_interno is not EAN). Dedup relies on name+brand.
func mapCoopArticulo(a coopArticulo) scrapingport.RawProduct {
	price := selectCoopPrice(a)
	return scrapingport.RawProduct{
		Title:    a.Descripcion,
		Brand:    a.MarcaDesc,
		Price:    &price,
		ImageURL: a.Imagen,
		SKU:      a.CodInterno,
		// EAN intentionally omitted — cod_interno is NOT an EAN.
		// Category is set by the source's category field (set at seed time per root category).
	}
}

// selectCoopPrice picks the best available price from the article.
// GOTCHA: all price fields are nullable JSON strings ("1234.00" or null).
// Priority: precio → precio_promo → precio_anterior.
func selectCoopPrice(a coopArticulo) float64 {
	if p := parseCoopPrice(a.Precio); p > 0 {
		return p
	}
	if p := parseCoopPrice(a.PrecioPromo); p > 0 {
		return p
	}
	return parseCoopPrice(a.PrecioAnterior)
}
