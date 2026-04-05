package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock source repo ---

type mockSourceRepo struct {
	sources            map[string]*sourceentity.Source
	recordJobResultCalls []recordJobResultCall
	updateNextRunAtCalls []updateNextRunAtCall
}

type recordJobResultCall struct {
	SourceID uuid.UUID
	Success  bool
}

type updateNextRunAtCall struct {
	SourceID  uuid.UUID
	NextRunAt time.Time
}

func newMockSourceRepo() *mockSourceRepo {
	return &mockSourceRepo{sources: make(map[string]*sourceentity.Source)}
}

func (m *mockSourceRepo) Save(ctx context.Context, s *sourceentity.Source) error {
	m.sources[s.ID.String()] = s
	return nil
}

func (m *mockSourceRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*sourceentity.Source, error) {
	s, ok := m.sources[id.String()]
	if !ok {
		return nil, errors.New("source not found")
	}
	return s, nil
}

func (m *mockSourceRepo) FindAll(ctx context.Context, tenantID uuid.UUID, filter sourceport.SourceFilter) ([]*sourceentity.Source, int, error) {
	return nil, 0, nil
}

func (m *mockSourceRepo) Update(ctx context.Context, s *sourceentity.Source) error {
	m.sources[s.ID.String()] = s
	return nil
}

func (m *mockSourceRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.sources, id.String())
	return nil
}

func (m *mockSourceRepo) FindDueForScraping(ctx context.Context) ([]*sourceentity.Source, error) {
	return nil, nil
}

func (m *mockSourceRepo) UpdateNextRunAt(ctx context.Context, sourceID uuid.UUID, nextRunAt time.Time) error {
	m.updateNextRunAtCalls = append(m.updateNextRunAtCalls, updateNextRunAtCall{SourceID: sourceID, NextRunAt: nextRunAt})
	s, ok := m.sources[sourceID.String()]
	if !ok {
		return errors.New("source not found")
	}
	s.NextRunAt = &nextRunAt
	return nil
}

func (m *mockSourceRepo) RecordJobResult(ctx context.Context, sourceID uuid.UUID, success bool) error {
	m.recordJobResultCalls = append(m.recordJobResultCalls, recordJobResultCall{SourceID: sourceID, Success: success})
	return nil
}

// --- Mock job repo ---

type mockJobRepo struct {
	jobs map[string]*entity.ScrapingJob
}

func newMockJobRepo() *mockJobRepo {
	return &mockJobRepo{jobs: make(map[string]*entity.ScrapingJob)}
}

func (m *mockJobRepo) Save(ctx context.Context, j *entity.ScrapingJob) error {
	m.jobs[j.ID.String()] = j
	return nil
}

func (m *mockJobRepo) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapingJob, error) {
	j, ok := m.jobs[id.String()]
	if !ok {
		return nil, errors.New("job not found")
	}
	return j, nil
}

func (m *mockJobRepo) FindAll(ctx context.Context, tenantID uuid.UUID, filter scrapeport.JobFilter) ([]*entity.ScrapingJob, int, error) {
	var result []*entity.ScrapingJob
	for _, j := range m.jobs {
		result = append(result, j)
	}
	return result, len(result), nil
}

func (m *mockJobRepo) Update(ctx context.Context, j *entity.ScrapingJob) error {
	m.jobs[j.ID.String()] = j
	return nil
}

func (m *mockJobRepo) ClaimPendingJob(ctx context.Context) (*entity.ScrapingJob, error) {
	for _, j := range m.jobs {
		if j.Status.IsPending() {
			return j, nil
		}
	}
	return nil, errors.New("no pending jobs")
}

// --- Helper: build source ---

func buildSource(tenantID uuid.UUID) *sourceentity.Source {
	s, _ := sourceentity.NewSource(sourceentity.CreateSourceParams{
		TenantID:   tenantID,
		Name:       "Test Source",
		BaseURL:    "https://example.com",
		Category:   "supermercados",
		SourceType: "ecommerce",
		Priority:   "medium",
		Tier:       2,
	})
	return s
}

// --- T-SCR-A01: TriggerScrape ---

func TestTriggerScrapeUseCase_HappyPath(t *testing.T) {
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()
	source := buildSource(tenantID)
	_ = sourceRepo.Save(context.Background(), source)

	uc := usecase.NewTriggerScrapeUseCase(sourceRepo, jobRepo)
	resp, err := uc.Execute(context.Background(), tenantID, source.ID)

	require.NoError(t, err)
	assert.Equal(t, source.ID, resp.SourceID)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, "manual", resp.TriggerType)
}

func TestTriggerScrapeUseCase_SourceNotFound(t *testing.T) {
	sourceRepo := newMockSourceRepo()
	jobRepo := newMockJobRepo()
	uc := usecase.NewTriggerScrapeUseCase(sourceRepo, jobRepo)

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source not found")
}

// --- T-SCR-A03: CancelJob (only pending) ---

func TestCancelJobUseCase_PendingJob(t *testing.T) {
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()
	job, _ := entity.NewScrapingJob(tenantID, uuid.New(), "manual")
	_ = jobRepo.Save(context.Background(), job)

	uc := usecase.NewCancelJobUseCase(jobRepo)
	err := uc.Execute(context.Background(), tenantID, job.ID)
	require.NoError(t, err)

	saved, _ := jobRepo.FindByID(context.Background(), tenantID, job.ID)
	assert.Equal(t, "cancelled", saved.Status.Value())
}

func TestCancelJobUseCase_NonPendingJobFails(t *testing.T) {
	jobRepo := newMockJobRepo()
	tenantID := uuid.New()
	job, _ := entity.NewScrapingJob(tenantID, uuid.New(), "manual")
	_ = job.Start()
	_ = jobRepo.Save(context.Background(), job)

	uc := usecase.NewCancelJobUseCase(jobRepo)
	err := uc.Execute(context.Background(), tenantID, job.ID)
	assert.Error(t, err)
}
