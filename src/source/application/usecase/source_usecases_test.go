package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/application/request"
	"github.com/mercadocercano/webdata-service/src/source/application/usecase"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock repo ---

type mockSourceRepo struct {
	sources map[string]*entity.Source
}

func newMockSourceRepo() *mockSourceRepo {
	return &mockSourceRepo{sources: make(map[string]*entity.Source)}
}

func (m *mockSourceRepo) Save(ctx context.Context, s *entity.Source) error {
	m.sources[s.ID.String()] = s
	return nil
}

func (m *mockSourceRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Source, error) {
	s, ok := m.sources[id.String()]
	if !ok {
		return nil, errors.New("source not found")
	}
	return s, nil
}

func (m *mockSourceRepo) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.SourceFilter) ([]*entity.Source, int, error) {
	var result []*entity.Source
	for _, s := range m.sources {
		result = append(result, s)
	}
	return result, len(result), nil
}

func (m *mockSourceRepo) Update(ctx context.Context, s *entity.Source) error {
	if _, ok := m.sources[s.ID.String()]; !ok {
		return errors.New("source not found")
	}
	m.sources[s.ID.String()] = s
	return nil
}

func (m *mockSourceRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.sources, id.String())
	return nil
}

func (m *mockSourceRepo) FindDueForScraping(ctx context.Context) ([]*entity.Source, error) {
	return nil, nil
}

// --- Helper ---

func makeCreateReq(name string) request.CreateSourceRequest {
	return request.CreateSourceRequest{
		Name:       name,
		BaseURL:    "https://example.com",
		Category:   "supermercados",
		SourceType: "ecommerce",
		Priority:   "medium",
		Tier:       2,
	}
}

// --- T-SRC-A01: CreateSource ---

func TestCreateSourceUseCase_HappyPath(t *testing.T) {
	repo := newMockSourceRepo()
	uc := usecase.NewCreateSourceUseCase(repo)
	tenantID := uuid.New()

	resp, err := uc.Execute(context.Background(), tenantID, makeCreateReq("Super Vea"))

	require.NoError(t, err)
	assert.Equal(t, "Super Vea", resp.Name)
	assert.Equal(t, tenantID, resp.TenantID)
}

func TestCreateSourceUseCase_DuplicateName(t *testing.T) {
	repo := newMockSourceRepo()
	uc := usecase.NewCreateSourceUseCase(repo)
	tenantID := uuid.New()

	_, err := uc.Execute(context.Background(), tenantID, makeCreateReq("Super Vea"))
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), tenantID, makeCreateReq("Super Vea"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// --- T-SRC-A02: UpdateSource ---

func TestUpdateSourceUseCase_PartialUpdate(t *testing.T) {
	repo := newMockSourceRepo()
	createUC := usecase.NewCreateSourceUseCase(repo)
	updateUC := usecase.NewUpdateSourceUseCase(repo)
	tenantID := uuid.New()

	created, err := createUC.Execute(context.Background(), tenantID, makeCreateReq("Carrefour"))
	require.NoError(t, err)

	newCity := "Resistencia"
	updated, err := updateUC.Execute(context.Background(), tenantID, created.ID, request.UpdateSourceRequest{
		City: &newCity,
	})

	require.NoError(t, err)
	assert.Equal(t, "Resistencia", updated.City)
	assert.Equal(t, "Carrefour", updated.Name)
}

func TestUpdateSourceUseCase_NotFound(t *testing.T) {
	repo := newMockSourceRepo()
	uc := usecase.NewUpdateSourceUseCase(repo)

	newName := "X"
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), request.UpdateSourceRequest{Name: &newName})
	assert.Error(t, err)
}

// --- T-SRC-A03: ListSources ---

func TestListSourcesUseCase_Filters(t *testing.T) {
	repo := newMockSourceRepo()
	createUC := usecase.NewCreateSourceUseCase(repo)
	listUC := usecase.NewListSourcesUseCase(repo)
	tenantID := uuid.New()

	_, _ = createUC.Execute(context.Background(), tenantID, makeCreateReq("Source A"))
	_, _ = createUC.Execute(context.Background(), tenantID, makeCreateReq("Source B"))

	result, err := listUC.Execute(context.Background(), tenantID, port.SourceFilter{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Sources, 2)
}
