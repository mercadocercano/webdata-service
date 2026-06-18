package dalle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/infrastructure/sources/dalle"
)

func buildDALLEResponse(imageURL string) map[string]interface{} {
	return map[string]interface{}{
		"data": []map[string]string{
			{"url": imageURL},
		},
	}
}

// Path principal de gpt-image-1: devuelve b64_json → el adapter retorna un data URI.
func TestDALLEGeneratorSource_B64JSON_ReturnsDataURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{{"b64_json": "aG9sYQ=="}},
		})
	}))
	defer server.Close()

	adapter := dalle.NewDALLEGeneratorSourceWithURL(server.URL, "test-key")
	result, err := adapter.FindByName(context.Background(), "Asado de tira")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result == nil || result.ImageURL != "data:image/png;base64,aG9sYQ==" {
		t.Errorf("esperaba data URI, got %+v", result)
	}
	if result.Source != entity.SourceDALLE {
		t.Errorf("esperaba source dalle, got %q", result.Source)
	}
}

func TestDALLEGeneratorSource_FindByName_GeneratesImage(t *testing.T) {
	// Arrange
	expectedURL := "https://oaidalleapiprodscus.blob.core.windows.net/private/img-abc123.png"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildDALLEResponse(expectedURL))
	}))
	defer server.Close()

	adapter := dalle.NewDALLEGeneratorSourceWithURL(server.URL, "test-key")

	// Act
	result, err := adapter.FindByName(context.Background(), "Coca-Cola 500ml lata")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected ProductData, got nil")
	}
	if result.ImageURL != expectedURL {
		t.Errorf("expected ImageURL %q, got %q", expectedURL, result.ImageURL)
	}
	if result.Source != entity.SourceDALLE {
		t.Errorf("expected source %q, got %q", entity.SourceDALLE, result.Source)
	}
}

func TestDALLEGeneratorSource_EmptyKey_ReturnsNil(t *testing.T) {
	// Arrange
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := dalle.NewDALLEGeneratorSourceWithURL(server.URL, "")

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

func TestDALLEGeneratorSource_APIError_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := dalle.NewDALLEGeneratorSourceWithURL(server.URL, "test-key")

	// Act
	result, err := adapter.FindByName(context.Background(), "producto test")

	// Assert
	if err == nil {
		t.Fatal("expected error from API 500, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got: %+v", result)
	}
}

func TestDALLEGeneratorSource_FindByGTIN_ReturnsNil(t *testing.T) {
	// Arrange
	adapter := dalle.NewDALLEGeneratorSourceWithURL("http://unused", "test-key")

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

func TestDALLEGeneratorSource_SourceName(t *testing.T) {
	adapter := dalle.NewDALLEGeneratorSourceWithURL("http://unused", "key")

	if adapter.SourceName() != entity.SourceDALLE {
		t.Errorf("expected SourceDALLE, got %q", adapter.SourceName())
	}
}
