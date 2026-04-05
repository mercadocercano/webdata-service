package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TEST-ID: T-SCR-D01

func aNewJob() *entity.ScrapingJob {
	j, _ := entity.NewScrapingJob(uuid.New(), uuid.New(), "manual")
	return j
}

func TestScrapingJob_Create_WithValidParams_Succeeds(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()

	job, err := entity.NewScrapingJob(tenantID, sourceID, "manual")

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, tenantID, job.TenantID)
	assert.Equal(t, sourceID, job.SourceID)
	assert.Equal(t, "pending", job.Status.Value())
	assert.Equal(t, "manual", job.TriggerType.Value())
	assert.Equal(t, 0, job.RetryCount)
	assert.Equal(t, 3, job.MaxRetries)
}

func TestScrapingJob_Create_WithInvalidTriggerType_ReturnsError(t *testing.T) {
	_, err := entity.NewScrapingJob(uuid.New(), uuid.New(), "unknown")
	assert.Error(t, err)
}

func TestScrapingJob_StateTransition_PendingToRunning(t *testing.T) {
	job := aNewJob()

	err := job.Start()

	require.NoError(t, err)
	assert.Equal(t, "running", job.Status.Value())
	assert.NotNil(t, job.StartedAt)
}

func TestScrapingJob_StateTransition_RunningToCompleted(t *testing.T) {
	job := aNewJob()
	_ = job.Start()

	err := job.Complete(10, 8)

	require.NoError(t, err)
	assert.Equal(t, "completed", job.Status.Value())
	assert.Equal(t, 10, job.ProductsFound)
	assert.Equal(t, 8, job.ProductsSaved)
	assert.NotNil(t, job.CompletedAt)
}

func TestScrapingJob_StateTransition_RunningToFailed(t *testing.T) {
	job := aNewJob()
	_ = job.Start()

	err := job.Fail("connection refused")

	require.NoError(t, err)
	assert.Equal(t, "failed", job.Status.Value())
	assert.Equal(t, "connection refused", job.ErrorMessage)
	assert.NotNil(t, job.CompletedAt)
}

func TestScrapingJob_StateTransition_PendingToCancelled(t *testing.T) {
	job := aNewJob()

	err := job.Cancel()

	require.NoError(t, err)
	assert.Equal(t, "cancelled", job.Status.Value())
}

func TestScrapingJob_CannotStartIfNotPending(t *testing.T) {
	job := aNewJob()
	_ = job.Start()

	err := job.Start()

	assert.Error(t, err)
}

func TestScrapingJob_CannotCompleteIfNotRunning(t *testing.T) {
	job := aNewJob()

	err := job.Complete(0, 0)

	assert.Error(t, err)
}

func TestScrapingJob_CannotCancelIfRunning(t *testing.T) {
	job := aNewJob()
	_ = job.Start()

	err := job.Cancel()

	assert.Error(t, err)
}

func TestScrapingJob_CanRetry_WhenFailedAndUnderMaxRetries(t *testing.T) {
	job := aNewJob()
	_ = job.Start()
	_ = job.Fail("error")

	assert.True(t, job.CanRetry())
}

func TestScrapingJob_CanRetry_False_WhenAtMaxRetries(t *testing.T) {
	job := aNewJob()
	job.RetryCount = 3

	assert.False(t, job.CanRetry())
}

// TEST-ID: T-SCR-D02
func TestJobStatus_ValidValues(t *testing.T) {
	for _, v := range []string{"pending", "running", "completed", "failed", "cancelled"} {
		_, err := entity.NewJobStatusFromString(v)
		assert.NoError(t, err)
	}
}

func TestJobStatus_InvalidValue_ReturnsError(t *testing.T) {
	_, err := entity.NewJobStatusFromString("unknown")
	assert.Error(t, err)
}

// --- Paginated job creation ---

func TestNewPaginatedScrapingJob_SetsPageField(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()

	job, err := entity.NewPaginatedScrapingJob(tenantID, sourceID, "scheduled", 3)

	require.NoError(t, err)
	assert.Equal(t, 3, job.Page)
	assert.Equal(t, tenantID, job.TenantID)
	assert.Equal(t, sourceID, job.SourceID)
	assert.Equal(t, "pending", job.Status.Value())
}

func TestNewPaginatedScrapingJob_InvalidTrigger_ReturnsError(t *testing.T) {
	_, err := entity.NewPaginatedScrapingJob(uuid.New(), uuid.New(), "invalid", 1)
	assert.Error(t, err)
}

func TestNewScrapingJob_PageDefaultsToZero(t *testing.T) {
	job, err := entity.NewScrapingJob(uuid.New(), uuid.New(), "manual")

	require.NoError(t, err)
	assert.Equal(t, 0, job.Page)
}
