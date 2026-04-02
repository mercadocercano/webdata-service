package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TEST-ID: T-SRC-D01

func aValidSource() entity.CreateSourceParams {
	return entity.CreateSourceParams{
		TenantID:        uuid.New(),
		Name:            "La Anónima Online",
		BaseURL:         "https://www.laanonima.com.ar",
		Category:        "supermercado",
		SourceType:      "ecommerce",
		City:            "Resistencia",
		Priority:        "medium",
		Tier:            2,
		FirecrawlMethod: "extract",
	}
}

func TestSource_Create_WithValidParams_Succeeds(t *testing.T) {
	params := aValidSource()

	s, err := entity.NewSource(params)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Equal(t, params.TenantID, s.TenantID)
	assert.Equal(t, params.Name, s.Name)
	assert.Equal(t, params.BaseURL, s.BaseURL)
	assert.True(t, s.IsActive)
	assert.InDelta(t, 1.00, s.HealthScore, 0.001)
	assert.Equal(t, 0, s.TotalRuns)
}

func TestSource_Create_WithEmptyName_ReturnsError(t *testing.T) {
	params := aValidSource()
	params.Name = ""

	_, err := entity.NewSource(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestSource_Create_WithEmptyBaseURL_ReturnsError(t *testing.T) {
	params := aValidSource()
	params.BaseURL = ""

	_, err := entity.NewSource(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestSource_Create_WithInvalidTier_ReturnsError(t *testing.T) {
	params := aValidSource()
	params.Tier = 5

	_, err := entity.NewSource(params)

	assert.Error(t, err)
}

func TestSource_Create_WithInvalidPriority_ReturnsError(t *testing.T) {
	params := aValidSource()
	params.Priority = "urgent"

	_, err := entity.NewSource(params)

	assert.Error(t, err)
}

func TestSource_RecordSuccess_UpdatesCounters(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	s.RecordSuccess()

	assert.Equal(t, 1, s.TotalRuns)
	assert.Equal(t, 1, s.SuccessfulRuns)
	assert.Equal(t, 0, s.FailedRuns)
	assert.NotNil(t, s.LastRunAt)
	assert.NotNil(t, s.LastSuccessAt)
}

func TestSource_RecordFailure_UpdatesCountersAndHealthScore(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	s.RecordFailure("connection timeout")

	assert.Equal(t, 1, s.TotalRuns)
	assert.Equal(t, 0, s.SuccessfulRuns)
	assert.Equal(t, 1, s.FailedRuns)
	assert.Equal(t, "connection timeout", s.LastFailureReason)
	assert.NotNil(t, s.LastRunAt)
}

func TestSource_HealthScore_DecreasesOnFailures(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	s.RecordSuccess()
	s.RecordFailure("error")
	s.RecordFailure("error")

	// 1 success, 2 failures out of 3 runs → health = 1/3 ≈ 0.33
	assert.Less(t, s.HealthScore, 1.00)
}

func TestSource_RecordFailure_SetsExponentialBackoff(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	s.RecordFailure("firecrawl API error 429")

	// Primer fallo: backoff = 5min * 2^0 = 5min
	assert.NotNil(t, s.NextRunAt)
	assert.True(t, s.NextRunAt.After(s.UpdatedAt))
	assert.Equal(t, 1, s.ConsecutiveFailures)
	assert.True(t, s.IsActive, "no debe desactivarse con el primer fallo")
}

func TestSource_RecordFailure_CircuitBreaker_DeactivatesAfterThreshold(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	for i := 0; i < 5; i++ {
		s.RecordFailure("firecrawl API error 402: payment required")
	}

	assert.False(t, s.IsActive, "debe desactivarse tras 5 fallos consecutivos")
	assert.Equal(t, 5, s.ConsecutiveFailures)
}

func TestSource_RecordSuccess_ResetConsecutiveFailures(t *testing.T) {
	params := aValidSource()
	s, _ := entity.NewSource(params)

	s.RecordFailure("error")
	s.RecordFailure("error")
	s.RecordSuccess()

	assert.Equal(t, 0, s.ConsecutiveFailures)
}
