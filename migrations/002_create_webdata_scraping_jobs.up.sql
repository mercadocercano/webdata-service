CREATE TABLE webdata_scraping_jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    source_id        UUID NOT NULL REFERENCES webdata_sources(id),
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    trigger_type     VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    firecrawl_job_id VARCHAR(255),
    products_found   INT DEFAULT 0,
    products_saved   INT DEFAULT 0,
    error_message    TEXT,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retry_count      SMALLINT NOT NULL DEFAULT 0,
    max_retries      SMALLINT NOT NULL DEFAULT 3
);

CREATE INDEX idx_jobs_tenant_status ON webdata_scraping_jobs(tenant_id, status);
CREATE INDEX idx_jobs_source ON webdata_scraping_jobs(source_id, created_at DESC);
