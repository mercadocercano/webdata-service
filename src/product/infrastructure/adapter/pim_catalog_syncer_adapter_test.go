package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
)

// newTestAdapter apunta el adapter al servidor httptest vía PIM_SERVICE_URL.
func newTestAdapter(t *testing.T, srvURL string) *PIMCatalogSyncerAdapter {
	t.Helper()
	t.Setenv("PIM_SERVICE_URL", srvURL)
	return NewPIMCatalogSyncerAdapter()
}

// SearchByNameBrand debe usar los params que PIM entiende (`search`/`brand`),
// NO los viejos `search_name`/`search_brand` que PIM ignoraba (bug del falso-positivo).
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

// PIM hace match por substring (ILIKE): puede devolver productos parecidos pero
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

// Update debe usar PUT (PIM expone PUT /global-catalog/products/:id; PATCH solo
// existe para /verify y /unverify → daba 404).
func TestUpdate_UsesPutMethod(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
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
	if gotMethod != http.MethodPut {
		t.Errorf("esperaba PUT, got %s", gotMethod)
	}
	if gotPath != "/api/v1/global-catalog/products/"+id.String() {
		t.Errorf("path inesperado: %s", gotPath)
	}
}
