package logging

import (
	"io"

	"github.com/mercadocercano/webdata-service/src/webdata/domain/port"

	sharedlog "github.com/hornosg/go-shared/infrastructure/logging"
)

// WebdataLogger implementa port.WebdataEventLogger emitiendo una línea JSON canónica
// (ADR-001) por evento, delegando el envelope (ts/level/service/event + campos flat
// omitempty) en go-shared CanonicalLogger (>= v0.8.0). El mapeo struct→fields y las
// reglas de nivel por evento viven acá; el formato canónico es compartido por la flota.
type WebdataLogger struct {
	canonical *sharedlog.CanonicalLogger
}

// NewWebdataLogger crea el adapter escribiendo a stdout. El service se fija acá, nunca por-call.
func NewWebdataLogger(service string) *WebdataLogger {
	return &WebdataLogger{canonical: sharedlog.NewCanonicalLogger(service)}
}

// NewWebdataLoggerWithWriter permite inyectar un io.Writer (tests).
func NewWebdataLoggerWithWriter(service string, w io.Writer) *WebdataLogger {
	return &WebdataLogger{canonical: sharedlog.NewCanonicalLoggerWithWriter(service, w)}
}

// levelFor aplica las reglas de nivel del ADR-001 por tipo de evento.
func levelFor(event string) string {
	switch event {
	// info: flujo OK — scraping, ingesta, enrichment completados
	case "webdata.scrape_completed",
		"webdata.job_completed",
		"webdata.products_ingested",
		"webdata.products_filtered",
		"webdata.product_business_type_assigned",
		"webdata.pim_sync_completed",
		"webdata.enrichment_completed",
		"webdata.scheduler_jobs_created":
		return "info"

	// warn: anomalías recuperables — fallos de persistencia individual, marca-sync fallida,
	// backoff de fuente, cron inválido/vacío, lote sin progreso
	case "webdata.scrape_failed",
		"webdata.job_failed",
		"webdata.product_save_failed",
		"webdata.product_business_type_failed",
		"webdata.pim_sync_failed",
		"webdata.pim_sync_mark_failed",
		"webdata.pim_refresh_failed",
		"webdata.source_backoff",
		"webdata.scheduler_source_error",
		"webdata.scheduler_job_error",
		"webdata.scheduler_cron_fallback",
		"webdata.worker_claim_error",
		"webdata.worker_execute_error",
		"webdata.enrichment_sources_failed",
		"webdata.enrichment_job_failed",
		"webdata.sync_batch_stalled":
		return "warn"

	// error: violación de estado — circuit breaker activa (fuente desactivada)
	case "webdata.source_circuit_breaker":
		return "error"

	default:
		return "info"
	}
}

func (l *WebdataLogger) Log(e port.WebdataEvent) {
	fields := map[string]any{
		"tenant_id":        e.TenantID,
		"source_id":        e.SourceID,
		"job_id":           e.JobID,
		"source_name":      e.SourceName,
		"business_type":    e.BusinessType,
		"firecrawl_method": e.FirecrawlMethod,
		"reason":           e.Reason,
		"duration":         e.Duration,
	}
	if e.ProductCount > 0 {
		fields["product_count"] = e.ProductCount
	}
	if e.BatchSize > 0 {
		fields["batch_size"] = e.BatchSize
	}
	if e.JobsCreated > 0 {
		fields["jobs_created"] = e.JobsCreated
	}
	if e.Synced > 0 {
		fields["synced"] = e.Synced
	}
	if e.Failed > 0 {
		fields["failed"] = e.Failed
	}
	if e.Skipped > 0 {
		fields["skipped"] = e.Skipped
	}
	if e.Page > 0 {
		fields["page"] = e.Page
	}
	l.canonical.Emit(levelFor(e.Event), e.Event, fields)
}
