package perplexity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/perplexity"
)

// mockMLSource implementa port.ProductSource para tests.
type mockMLSource struct {
	data *entity.ProductData
	err  error
}

func (m *mockMLSource) FindByGTIN(_ context.Context, _ string) (*entity.ProductData, error) {
	return m.data, m.err
}

func (m *mockMLSource) FindByName(_ context.Context, _ string) (*entity.ProductData, error) {
	return m.data, m.err
}

func (m *mockMLSource) FindByNameWithContext(_ context.Context, _, _, _ string) (*entity.ProductData, error) {
	return m.data, m.err
}

func (m *mockMLSource) SourceName() entity.Source {
	return entity.SourceML
}

func buildPerplexityResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]string{
					"content": content,
				},
			},
		},
	}
}

func TestPerplexityGTINSource_FindByName_ValidGTIN_ReturnsImage(t *testing.T) {
	// Arrange
	// 7750226123456: check digit validation
	// sum(x1/x3 alternating pos 0..11): 7*1+7*3+5*1+0*3+2*1+2*3+6*1+1*3+2*1+3*3+4*1+5*3
	// = 7+21+5+0+2+6+6+3+2+9+4+15 = 80 → (10 - 80%10)%10 = 0 → last digit must be 0
	// Usamos un GTIN-13 con check digit correcto: 4006381333931
	validGTIN := "4006381333931"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildPerplexityResponse(validGTIN))
	}))
	defer server.Close()

	expectedData := &entity.ProductData{
		ImageURL: "https://example.com/image.jpg",
		Source:   entity.SourceML,
	}

	ml := &mockMLSource{data: expectedData}
	adapter := perplexity.NewPerplexityGTINSourceWithURL(server.URL, "test-key", ml)

	// Act
	result, err := adapter.FindByName(context.Background(), "Coca-Cola 500ml")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected ProductData, got nil")
	}
	if result.ImageURL != expectedData.ImageURL {
		t.Errorf("expected ImageURL %q, got %q", expectedData.ImageURL, result.ImageURL)
	}
}

func TestPerplexityGTINSource_FindByName_InvalidGTIN_ReturnsNil(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildPerplexityResponse("123"))
	}))
	defer server.Close()

	ml := &mockMLSource{}
	adapter := perplexity.NewPerplexityGTINSourceWithURL(server.URL, "test-key", ml)

	// Act
	result, err := adapter.FindByName(context.Background(), "producto desconocido")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestPerplexityGTINSource_EmptyKey_ReturnsNil(t *testing.T) {
	// Arrange — servidor que fallaría si fuera llamado
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ml := &mockMLSource{}
	adapter := perplexity.NewPerplexityGTINSourceWithURL(server.URL, "", ml)

	// Act
	result, err := adapter.FindByName(context.Background(), "cualquier producto")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
	if called {
		t.Error("expected HTTP server NOT to be called when key is empty")
	}
}

func TestPerplexityGTINSource_FindByGTIN_ReturnsNil(t *testing.T) {
	// Arrange
	ml := &mockMLSource{}
	adapter := perplexity.NewPerplexityGTINSourceWithURL("http://unused", "test-key", ml)

	// Act
	result, err := adapter.FindByGTIN(context.Background(), "4006381333931")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestPerplexityGTINSource_SourceName(t *testing.T) {
	ml := &mockMLSource{}
	adapter := perplexity.NewPerplexityGTINSourceWithURL("http://unused", "key", ml)

	if adapter.SourceName() != entity.SourcePerplexity {
		t.Errorf("expected SourcePerplexity, got %q", adapter.SourceName())
	}
}
