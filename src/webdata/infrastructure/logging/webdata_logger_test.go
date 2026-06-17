package logging_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mercadocercano/webdata-service/src/webdata/domain/port"
	"github.com/mercadocercano/webdata-service/src/webdata/infrastructure/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebdataLogger_EmitsCanonicalEnvelope(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event:        "webdata.scrape_completed",
		TenantID:     "tenant-abc",
		SourceID:     "source-123",
		JobID:        "job-456",
		ProductCount: 42,
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	assert.Equal(t, "webdata.scrape_completed", line["event"])
	assert.Equal(t, "info", line["level"])
	assert.Equal(t, "webdata-test", line["service"])
	assert.NotEmpty(t, line["ts"])
	assert.Equal(t, "tenant-abc", line["tenant_id"])
	assert.Equal(t, "source-123", line["source_id"])
	assert.Equal(t, "job-456", line["job_id"])
	assert.EqualValues(t, 42, line["product_count"])
}

func TestWebdataLogger_LevelWarn_OnScrapeFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event:    "webdata.scrape_failed",
		SourceID: "source-999",
		Reason:   "timeout after 30s",
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	assert.Equal(t, "webdata.scrape_failed", line["event"])
	assert.Equal(t, "warn", line["level"])
	assert.Equal(t, "timeout after 30s", line["reason"])
	// product_count omitted when 0 (omitempty)
	assert.Nil(t, line["product_count"])
}

func TestWebdataLogger_LevelError_OnCircuitBreaker(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event:      "webdata.source_circuit_breaker",
		SourceID:   "source-777",
		SourceName: "Tienda XYZ",
		Reason:     "5 consecutive failures",
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	assert.Equal(t, "error", line["level"])
	assert.Equal(t, "Tienda XYZ", line["source_name"])
}

func TestWebdataLogger_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event: "webdata.products_ingested",
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	// campos vacíos no deben aparecer
	assert.Nil(t, line["tenant_id"])
	assert.Nil(t, line["source_id"])
	assert.Nil(t, line["reason"])
	assert.Nil(t, line["product_count"])
}

func TestWebdataLogger_EnrichmentCompleted(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event:        "webdata.enrichment_completed",
		BusinessType: "bebidas",
		JobsCreated:  7,
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	assert.Equal(t, "webdata.enrichment_completed", line["event"])
	assert.Equal(t, "info", line["level"])
	assert.Equal(t, "bebidas", line["business_type"])
	assert.EqualValues(t, 7, line["jobs_created"])
}

func TestWebdataLogger_PIMSyncCompleted(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewWebdataLoggerWithWriter("webdata-test", &buf)

	logger.Log(port.WebdataEvent{
		Event:  "webdata.pim_sync_completed",
		Synced: 15,
	})

	var line map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

	assert.Equal(t, "info", line["level"])
	assert.EqualValues(t, 15, line["synced"])
}
