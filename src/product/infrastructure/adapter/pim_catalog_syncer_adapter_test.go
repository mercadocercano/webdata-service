package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/shared/platformauth"
)

// newTestTokens construye un ServiceTokenProvider de prueba (D8) con un secreto fijo.
func newTestTokens(t *testing.T) *platformauth.ServiceTokenProvider {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-platformauth-32chars-min!!")
	tokens, err := platformauth.NewServiceTokenProvider()
	if err != nil {
		t.Fatalf("newTestTokens: %v", err)
	}
	return tokens
}

// newTestAdapter apunta el adapter al servidor httptest vía CATALOG_SERVICE_URL.
func newTestAdapter(t *testing.T, srvURL string) *PIMCatalogSyncerAdapter {
	t.Helper()
	t.Setenv("CATALOG_SERVICE_URL", srvURL)
	return NewPIMCatalogSyncerAdapter(newTestTokens(t))
}

// SearchByNameBrand debe usar los params que catalog-service entiende (`search`/`brand`),
// NO los viejos `search_name`/`search_brand` que se ignoraban (bug del falso-positivo).
func TestSearchByNameBrand_UsesSearchAndBrandParams(t *testing.T) {
	var gotSearch, gotBrand, gotLegacyName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotSearch = q.Get("search")
		gotBrand = q.Get("brand")
		gotLegacyName = q.Get("search_name")
		json.NewEncoder(w).Encode(pimGlobalProductListResponse{Items: []port.PIMGlobalProduct{}})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	_, err := a.SearchByNameBrand(context.Background(), uuid.New(), "Leche Entera", "La Serenisima")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSearch != "Leche Entera" {
		t.Errorf("expected search=%q, got %q", "Leche Entera", gotSearch)
	}
	if gotBrand != "La Serenisima" {
		t.Errorf("expected brand=%q, got %q", "La Serenisima", gotBrand)
	}
	if gotLegacyName != "" {
		t.Errorf("no debe mandar el param legacy search_name, got %q", gotLegacyName)
	}
}

// catalog hace match por substring (ILIKE): puede devolver productos parecidos pero
// distintos. El adapter solo debe aceptar match si el nombre es EXACTO
// (case-insensitive) — si no, devolver nil para que el sync cree el producto
// en vez de pisar uno real distinto.
func TestSearchByNameBrand_OnlyExactNameMatches(t *testing.T) {
	target := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Primer item es un substring-match irrelevante; el segundo es el exacto.
		json.NewEncoder(w).Encode(pimGlobalProductListResponse{Items: []port.PIMGlobalProduct{
			{ID: uuid.New(), Name: "Leche Entera Sachet Oferta"},
			{ID: target, Name: "leche entera"}, // exacto salvo el case
		}})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	got, err := a.SearchByNameBrand(context.Background(), uuid.New(), "Leche Entera", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("esperaba match exacto (case-insensitive), got nil")
	}
	if got.ID != target {
		t.Errorf("esperaba el item exacto %s, got %s (%q)", target, got.ID, got.Name)
	}
}

func TestSearchByNameBrand_NoExactMatchReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Solo substring-matches, ninguno exacto.
		json.NewEncoder(w).Encode(pimGlobalProductListResponse{Items: []port.PIMGlobalProduct{
			{ID: uuid.New(), Name: "Leche Entera Sachet 1L"},
			{ID: uuid.New(), Name: "Leche Entera Larga Vida"},
		}})
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	got, err := a.SearchByNameBrand(context.Background(), uuid.New(), "Leche Entera", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("esperaba nil (sin match exacto), got %q", got.Name)
	}
}

// Update debe usar PATCH (T6a expone PATCH, no PUT), autenticar con el JWT de servicio
// (D8), NO mandar X-User-Role (forjable) ni X-Tenant-ID, y SÍ mandar X-Request-Id
// (correlation_id obligatorio para la auditoría A09 del writer de T6a — sin él el
// PATCH devuelve 400, defecto detectado al correr el E2E del ciclo de enrichment).
func TestUpdate_UsesPatchAndServiceAuth(t *testing.T) {
	// Header de rol forjable, en minúsculas a propósito: Header.Get canonicaliza, y la
	// forma canónica literal sólo se admite en comentarios (check-hexagonal, regla 5).
	const forgedRoleHeader = "x-user-role"
	var gotMethod, gotPath, gotAuth, gotUserRole, gotTenantID, gotRequestID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotUserRole = r.Header.Get(forgedRoleHeader)
		gotTenantID = r.Header.Get("X-Tenant-ID")
		gotRequestID = r.Header.Get("X-Request-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := newTestAdapter(t, srv.URL)
	id := uuid.New()
	price := 1234.5
	err := a.Update(context.Background(), uuid.New(), id, port.UpdatePIMProductRequest{
		Price:    &price,
		ImageURL: "https://cdn.example.com/x.webp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("esperaba PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/global-catalog/products/"+id.String() {
		t.Errorf("path inesperado: %s", gotPath)
	}
	if gotAuth == "" || gotAuth[:7] != "Bearer " {
		t.Errorf("esperaba Authorization: Bearer <jwt>, got %q", gotAuth)
	}
	if gotUserRole != "" {
		t.Errorf("no debe mandar el header de rol (forjable, D3.1), got %q", gotUserRole)
	}
	if gotTenantID != "" {
		t.Errorf("no debe mandar X-Tenant-ID (catalog sin tenant), got %q", gotTenantID)
	}
	if gotRequestID == "" {
		t.Errorf("debe mandar X-Request-Id (A09 del writer T6a — sin él PATCH da 400), got vacío")
	}
}
