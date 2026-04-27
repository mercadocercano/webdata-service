package image

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

type PassthroughEnhancer struct {
	httpClient *http.Client
}

func NewPassthroughEnhancer() *PassthroughEnhancer {
	return &PassthroughEnhancer{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Enhance descarga la imagen original sin procesamiento adicional.
func (e *PassthroughEnhancer) Enhance(ctx context.Context, rawURL string, _ entity.Source) ([]byte, string, int, error) {
	imageBytes, err := downloadURL(ctx, e.httpClient, rawURL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("passthrough download: %w", err)
	}
	return imageBytes, detectMimeType(imageBytes), 0, nil
}

// detectMimeType infiere el mime type por magic bytes.
func detectMimeType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "image/gif"
	case data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46:
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
