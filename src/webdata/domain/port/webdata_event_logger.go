package port

// WebdataEvent es el payload canónico para eventos de dominio de webdata (ADR-001).
// Campos flat, named. Los nombres comunes (tenant_id) son idénticos al resto de la
// flota para que el LogQL cross-service funcione. Todos opcionales salvo Event.
//
// Regla PII: NUNCA incluir API keys, tokens, contraseñas ni datos personales.
type WebdataEvent struct {
	Event        string // <domain>.<action>_<result>, ej: "webdata.scrape_completed"
	TenantID     string
	SourceID     string
	JobID        string
	ProductCount int
	BusinessType string
	Reason       string
	SourceName   string
	// scraping
	FirecrawlMethod string
	// enrichment
	BatchSize     int
	JobsCreated   int
	// sync
	Synced  int
	Failed  int
	Skipped int
	// scheduler
	Page     int
	Duration string
}

// WebdataEventLogger es el puerto para emitir eventos canónicos de webdata.
// El código de aplicación depende de esta interfaz; el adapter (JSON a stdout,
// Loki push, etc.) la implementa. Nunca al revés.
type WebdataEventLogger interface {
	Log(e WebdataEvent)
}
