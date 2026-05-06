package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mercadocercano/webdata-service/src/enrichment/application"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockPIMClient struct {
	items  []port.NeedsEnrichmentItem
	total  int
	err    error
	updated map[string]port.UpdateProductRequest
}

func newMockPIMClient(items []port.NeedsEnrichmentItem) *mockPIMClient {
	return &mockPIMClient{items: items, total: len(items), updated: make(map[string]port.UpdateProductRequest)}
}

func (m *mockPIMClient) ListNeedingEnrichment(_ context.Context, _ string, _ int) (*port.NeedsEnrichmentResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &port.NeedsEnrichmentResult{Items: m.items, Total: m.total}, nil
}

func (m *mockPIMClient) ListNeedingEnrichmentPage(_ context.Context, _ string, _, _ int) (*port.NeedsEnrichmentResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &port.NeedsEnrichmentResult{Items: m.items, Total: m.total}, nil
}

func (m *mockPIMClient) GetProductsByIDs(_ context.Context, _ []string) (*port.NeedsEnrichmentResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &port.NeedsEnrichmentResult{Items: m.items, Total: m.total}, nil
}

func (m *mockPIMClient) UpdateProduct(_ context.Context, productID string, req port.UpdateProductRequest) error {
	m.updated[productID] = req
	return nil
}

type mockProductSource struct {
	sourceName entity.Source
	data       *entity.ProductData
	err        error
}

func (m *mockProductSource) FindByGTIN(_ context.Context, _ string) (*entity.ProductData, error) {
	return m.data, m.err
}

func (m *mockProductSource) FindByName(_ context.Context, _ string) (*entity.ProductData, error) {
	return m.data, m.err
}

func (m *mockProductSource) SourceName() entity.Source {
	return m.sourceName
}

type mockImageEnhancer struct {
	imageBytes []byte
	mimeType   string
	costCents  int
	err        error
}

func (m *mockImageEnhancer) Enhance(_ context.Context, _ string, _ entity.Source) ([]byte, string, int, error) {
	return m.imageBytes, m.mimeType, m.costCents, m.err
}

type mockImageStore struct {
	cdnURL string
	cdnKey string
	err    error
}

func (m *mockImageStore) Upload(_ context.Context, _ string, _ []byte, _ string) (string, string, error) {
	return m.cdnURL, m.cdnKey, m.err
}

func (m *mockImageStore) Exists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

type mockEnrichmentRepo struct {
	resolved map[string]bool
	saved    []*entity.ImageResolution
}

func newMockEnrichmentRepo() *mockEnrichmentRepo {
	return &mockEnrichmentRepo{resolved: make(map[string]bool)}
}

func (m *mockEnrichmentRepo) IsResolved(_ context.Context, gtin string) (bool, error) {
	return m.resolved[gtin], nil
}

func (m *mockEnrichmentRepo) Save(_ context.Context, res *entity.ImageResolution) error {
	m.saved = append(m.saved, res)
	return nil
}

func (m *mockEnrichmentRepo) GetStatus(_ context.Context) (*port.EnrichmentStatus, error) {
	return &port.EnrichmentStatus{TotalResolved: len(m.saved), BySource: make(map[string]int)}, nil
}

// --- Tests ---

func TestRunEnrichmentBatch_SkipsAlreadyResolved(t *testing.T) {
	repo := newMockEnrichmentRepo()
	repo.resolved["prod-1"] = true // skip check usa item.ID como clave estable

	pim := newMockPIMClient([]port.NeedsEnrichmentItem{
		{ID: "prod-1", EAN: "7790895000430", Name: "Coca-Cola"},
	})

	uc := application.NewRunEnrichmentBatchUseCase(
		pim,
		[]port.ProductSource{},
		&mockImageEnhancer{imageBytes: []byte("img"), mimeType: "image/jpeg"},
		&mockImageStore{cdnURL: "https://cdn.test/img.jpg", cdnKey: "products/key"},
		repo,
	)

	result, err := uc.Execute(context.Background(), application.RunEnrichmentBatchRequest{
		BusinessType: "supermercado",
		Limit:        10,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, repo.saved)
}

func TestRunEnrichmentBatch_MLHit_UploadsAndUpdatesPIM(t *testing.T) {
	repo := newMockEnrichmentRepo()
	pim := newMockPIMClient([]port.NeedsEnrichmentItem{
		{ID: "prod-1", EAN: "7790895000430", Name: "Coca-Cola"},
	})

	mlSource := &mockProductSource{
		sourceName: entity.SourceML,
		data: &entity.ProductData{
			EAN:      "7790895000430",
			Name:     "Coca-Cola Zero",
			Brand:    "Coca-Cola",
			Category: "SOFT_DRINKS",
			ImageURL: "https://ml.static.com/img.jpg",
			Source:   entity.SourceML,
		},
	}

	uc := application.NewRunEnrichmentBatchUseCase(
		pim,
		[]port.ProductSource{mlSource},
		&mockImageEnhancer{imageBytes: []byte("img"), mimeType: "image/jpeg", costCents: 0},
		&mockImageStore{cdnURL: "https://cdn.test/img.jpg", cdnKey: "products/7790895000430/600x600.webp"},
		repo,
	)

	result, err := uc.Execute(context.Background(), application.RunEnrichmentBatchRequest{
		BusinessType: "supermercado",
		Limit:        10,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 1, len(repo.saved))
	assert.Equal(t, "7790895000430", repo.saved[0].EAN)
	assert.Equal(t, entity.SourceML, repo.saved[0].Source)
	assert.NotEmpty(t, pim.updated["prod-1"].ImageURL)
}

func TestRunEnrichmentBatch_MLFail_OFFFallback(t *testing.T) {
	repo := newMockEnrichmentRepo()
	pim := newMockPIMClient([]port.NeedsEnrichmentItem{
		{ID: "prod-2", EAN: "12345678", Name: "Granola Bar"},
	})

	mlSource := &mockProductSource{sourceName: entity.SourceML, data: nil}
	offSource := &mockProductSource{
		sourceName: entity.SourceOFF,
		data: &entity.ProductData{
			EAN:      "12345678",
			Name:     "Granola Bar",
			Brand:    "Kelloggs",
			ImageURL: "https://off.images.com/img.jpg",
			Source:   entity.SourceOFF,
		},
	}

	uc := application.NewRunEnrichmentBatchUseCase(
		pim,
		[]port.ProductSource{mlSource, offSource},
		&mockImageEnhancer{imageBytes: []byte("img"), mimeType: "image/jpeg"},
		&mockImageStore{cdnURL: "https://cdn.test/img.jpg", cdnKey: "products/key"},
		repo,
	)

	result, err := uc.Execute(context.Background(), application.RunEnrichmentBatchRequest{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 1, len(repo.saved))
	assert.Equal(t, entity.SourceOFF, repo.saved[0].Source)
}

func TestRunEnrichmentBatch_AllSourcesFail_CountsInFailed(t *testing.T) {
	repo := newMockEnrichmentRepo()
	pim := newMockPIMClient([]port.NeedsEnrichmentItem{
		{ID: "prod-3", EAN: "99999999", Name: "Unknown Product"},
	})

	failSource := &mockProductSource{
		sourceName: entity.SourceML,
		err:        errors.New("ml unavailable"),
	}

	uc := application.NewRunEnrichmentBatchUseCase(
		pim,
		[]port.ProductSource{failSource},
		&mockImageEnhancer{imageBytes: []byte("img"), mimeType: "image/jpeg"},
		&mockImageStore{cdnURL: "https://cdn.test/img.jpg", cdnKey: "products/key"},
		repo,
	)

	result, err := uc.Execute(context.Background(), application.RunEnrichmentBatchRequest{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 1, result.Failed)
	assert.NotEmpty(t, result.Errors)
}

func TestRunEnrichmentBatch_ForceReprocess_IgnoresResolved(t *testing.T) {
	repo := newMockEnrichmentRepo()
	repo.resolved["7790895000430"] = true

	pim := newMockPIMClient([]port.NeedsEnrichmentItem{
		{ID: "prod-1", EAN: "7790895000430", Name: "Coca-Cola"},
	})

	mlSource := &mockProductSource{
		sourceName: entity.SourceML,
		data: &entity.ProductData{
			EAN:      "7790895000430",
			ImageURL: "https://ml.static.com/img.jpg",
			Source:   entity.SourceML,
		},
	}

	uc := application.NewRunEnrichmentBatchUseCase(
		pim,
		[]port.ProductSource{mlSource},
		&mockImageEnhancer{imageBytes: []byte("img"), mimeType: "image/jpeg"},
		&mockImageStore{cdnURL: "https://cdn.test/img.jpg", cdnKey: "products/key"},
		repo,
	)

	result, err := uc.Execute(context.Background(), application.RunEnrichmentBatchRequest{
		Limit:          10,
		ForceReprocess: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Processed)
	assert.Equal(t, 0, result.Skipped)
}
