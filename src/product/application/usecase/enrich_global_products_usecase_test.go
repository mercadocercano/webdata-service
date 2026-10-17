package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	scrapingentity "github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Fakes for the enrichment usecase (T11 / O2c) ────────────────────────────

type enrichPIMSyncer struct {
	items []port.NeedsEnrichmentItem
	total int
	err   error
}

func (s *enrichPIMSyncer) ListNeedingEnrichment(_ context.Context, _ string, _, _ int) (*port.ListNeedingEnrichmentResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &port.ListNeedingEnrichmentResult{Items: s.items, Total: s.total}, nil
}

// Stubs for the rest of port.PIMCatalogSyncer — unused by the enrichment usecase.
func (s *enrichPIMSyncer) SearchByEAN(context.Context, uuid.UUID, string) (*port.PIMGlobalProduct, error) {
	return nil, nil
}
func (s *enrichPIMSyncer) SearchByNameBrand(context.Context, uuid.UUID, string, string) (*port.PIMGlobalProduct, error) {
	return nil, nil
}
func (s *enrichPIMSyncer) Create(context.Context, uuid.UUID, port.CreatePIMProductRequest) (*port.PIMGlobalProduct, error) {
	return nil, nil
}
func (s *enrichPIMSyncer) Update(context.Context, uuid.UUID, uuid.UUID, port.UpdatePIMProductRequest) error {
	return nil
}
func (s *enrichPIMSyncer) RefreshTemplateProducts(context.Context) error { return nil }

// enrichSourceRepo records which read path the usecase takes. The whole point of T9c2 is
// that the enrichment batch reaches sources through the platform method, not the
// tenant-scoped FindAll with a sentinel tenant_id.
type enrichSourceRepo struct {
	platformSources map[string][]*sourceentity.Source // category -> sources
	platformCalls   []string
	findAllCalls    int // MUST stay 0 — the sentinel path is gone
}

func (r *enrichSourceRepo) FindAllActiveByCategoryPlatform(_ context.Context, category string) ([]*sourceentity.Source, error) {
	r.platformCalls = append(r.platformCalls, category)
	return r.platformSources[category], nil
}

func (r *enrichSourceRepo) FindAll(context.Context, uuid.UUID, sourceport.SourceFilter) ([]*sourceentity.Source, int, error) {
	r.findAllCalls++
	return nil, 0, nil
}

func (r *enrichSourceRepo) FindByID(context.Context, uuid.UUID, uuid.UUID) (*sourceentity.Source, error) {
	return nil, nil
}
func (r *enrichSourceRepo) Save(context.Context, *sourceentity.Source) error  { return nil }
func (r *enrichSourceRepo) Update(context.Context, *sourceentity.Source) error { return nil }
func (r *enrichSourceRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *enrichSourceRepo) FindDueForScraping(context.Context) ([]*sourceentity.Source, error) {
	return nil, nil
}
func (r *enrichSourceRepo) UpdateNextRunAt(context.Context, uuid.UUID, time.Time) error { return nil }
func (r *enrichSourceRepo) RecordJobResult(context.Context, uuid.UUID, bool) error      { return nil }

type enrichJobRepo struct {
	saved []*scrapingentity.ScrapingJob
}

func (r *enrichJobRepo) Save(_ context.Context, job *scrapingentity.ScrapingJob) error {
	r.saved = append(r.saved, job)
	return nil
}
func (r *enrichJobRepo) FindByID(context.Context, uuid.UUID, uuid.UUID) (*scrapingentity.ScrapingJob, error) {
	return nil, nil
}
func (r *enrichJobRepo) FindAll(context.Context, uuid.UUID, scrapingport.JobFilter) ([]*scrapingentity.ScrapingJob, int, error) {
	return nil, 0, nil
}
func (r *enrichJobRepo) Update(context.Context, *scrapingentity.ScrapingJob) error { return nil }
func (r *enrichJobRepo) ClaimPendingJob(context.Context) (*scrapingentity.ScrapingJob, error) {
	return nil, nil
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TEST-ID: T11 (PLAT-E39 T9c2)
// A full enrichment cycle: PIM returns products across two business types; the usecase
// finds active sources per business type through the EXPLICIT platform read path
// (FindAllActiveByCategoryPlatform), never the removed GlobalTenantID sentinel path
// (FindAll with a zero tenant_id). This is the T9c2 contribution: the read-path swap.
//
// NOTE on JobsCreated: the usecase's job-creation step (NewScrapingJob(..., "enrichment"))
// is gated by a PRE-EXISTING bug — "enrichment" is not a valid TriggerType (only
// scheduled/manual are, src/scraping/domain/value_object/trigger_type.go). That bug
// predates T9c2 (the usecase file is untouched by T9c1/T9b) and is out of T9c2's scope;
// it is recorded as a finding, not fixed here (fixing it would change behavior beyond
// the sentinel replacement). The assertion below pins the pre-existing 0-jobs behavior
// so a future fix is a visible delta, not a silent one.
func TestEnrichGlobalProducts_FindsSourcesViaPlatformPathNotSentinel(t *testing.T) {
	almacenSrc1 := &sourceentity.Source{ID: uuid.New(), Category: "almacen", IsActive: true}
	almacenSrc2 := &sourceentity.Source{ID: uuid.New(), Category: "almacen", IsActive: true}
	bebidasSrc1 := &sourceentity.Source{ID: uuid.New(), Category: "bebidas", IsActive: true}

	repo := &enrichSourceRepo{
		platformSources: map[string][]*sourceentity.Source{
			"almacen": {almacenSrc1, almacenSrc2},
			"bebidas": {bebidasSrc1},
		},
	}
	jobs := &enrichJobRepo{}
	syncer := &enrichPIMSyncer{
		items: []port.NeedsEnrichmentItem{
			{ID: "p1", BusinessType: "almacen"},
			{ID: "p2", BusinessType: "bebidas"},
		},
		total: 2,
	}

	uc := NewEnrichGlobalProductsUseCase(syncer, repo, jobs)
	res, err := uc.Execute(context.Background(), "almacen")
	require.NoError(t, err)

	assert.Equal(t, 2, res.ProductsNeedingEnrichment)
	// T9c2 core: sources are reached through the platform read path, one call per business type.
	assert.ElementsMatch(t, []string{"almacen", "bebidas"}, repo.platformCalls,
		"FindAllActiveByCategoryPlatform was invoked once per business type")
	// T9c2 core: the tenant-scoped FindAll (former sentinel path) is NOT used.
	assert.Equal(t, 0, repo.findAllCalls,
		"tenant-scoped FindAll must NOT be called — the GlobalTenantID sentinel path is gone")
	// Pre-existing behavior, pinned (see NOTE above): no jobs are saved because "enrichment"
	// is not a valid TriggerType. This is the same count as before T9c2.
	assert.Equal(t, 0, res.JobsCreated, "pre-existing: NewScrapingJob rejects \"enrichment\" TriggerType")
	assert.Empty(t, jobs.saved)
}

// TEST-ID: T11b (PLAT-E39 T9c2)
// A business type with no active sources reaches the platform path, creates no jobs, no error.
func TestEnrichGlobalProducts_NoSourcesForBusinessType(t *testing.T) {
	repo := &enrichSourceRepo{platformSources: map[string][]*sourceentity.Source{}}
	jobs := &enrichJobRepo{}
	syncer := &enrichPIMSyncer{
		items: []port.NeedsEnrichmentItem{{ID: "p1", BusinessType: "sin-fuentes"}},
		total: 1,
	}

	uc := NewEnrichGlobalProductsUseCase(syncer, repo, jobs)
	res, err := uc.Execute(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 0, res.JobsCreated)
	assert.Empty(t, jobs.saved)
	assert.Equal(t, []string{"sin-fuentes"}, repo.platformCalls)
	assert.Equal(t, 0, repo.findAllCalls)
}