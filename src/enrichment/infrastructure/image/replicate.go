package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
)

// ErrNotConfigured indica que el enhancer no está configurado.
var ErrNotConfigured = errors.New("replicate: REPLICATE_API_TOKEN not set")

const (
	replicateCostCents  = 20
	replicatePollSleep  = 3 * time.Second
	replicateMaxWait    = 60 * time.Second
)

type ReplicateEnhancer struct {
	apiToken   string
	httpClient *http.Client
}

func NewReplicateEnhancer() *ReplicateEnhancer {
	return &ReplicateEnhancer{
		apiToken:   os.Getenv("REPLICATE_API_TOKEN"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *ReplicateEnhancer) Enhance(ctx context.Context, rawURL string, _ entity.Source) ([]byte, string, int, error) {
	if e.apiToken == "" {
		return nil, "", 0, ErrNotConfigured
	}

	predictionID, err := e.createPrediction(ctx, rawURL)
	if err != nil {
		return nil, "", 0, err
	}

	outputURL, err := e.pollUntilDone(ctx, predictionID)
	if err != nil {
		return nil, "", 0, err
	}

	imageBytes, err := downloadURL(ctx, e.httpClient, outputURL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("replicate downloading output: %w", err)
	}

	return imageBytes, "image/png", replicateCostCents, nil
}

func (e *ReplicateEnhancer) createPrediction(ctx context.Context, imageURL string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"input": map[string]string{"image": imageURL},
	})
	if err != nil {
		return "", fmt.Errorf("replicate marshaling: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.replicate.com/v1/models/lucataco/remove-bg/predictions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("replicate creating request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+e.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("replicate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("replicate returned status %d on create", resp.StatusCode)
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("replicate decoding prediction: %w", err)
	}
	return body.ID, nil
}

func (e *ReplicateEnhancer) pollUntilDone(ctx context.Context, predictionID string) (string, error) {
	deadline := time.Now().Add(replicateMaxWait)
	pollURL := fmt.Sprintf("https://api.replicate.com/v1/predictions/%s", predictionID)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(replicatePollSleep):
		}

		output, done, err := e.checkPrediction(ctx, pollURL)
		if err != nil {
			return "", err
		}
		if done {
			return output, nil
		}
	}

	return "", fmt.Errorf("replicate prediction %s timed out after %s", predictionID, replicateMaxWait)
}

func (e *ReplicateEnhancer) checkPrediction(ctx context.Context, pollURL string) (output string, done bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return "", false, fmt.Errorf("replicate poll request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+e.apiToken)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("replicate poll failed: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		Status string `json:"status"`
		Output string `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", false, fmt.Errorf("replicate decoding poll: %w", err)
	}

	switch body.Status {
	case "succeeded":
		return body.Output, true, nil
	case "failed", "canceled":
		return "", false, fmt.Errorf("replicate prediction ended with status: %s", body.Status)
	}
	return "", false, nil
}

func downloadURL(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
