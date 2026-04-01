CREATE TABLE webdata_sources (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(255) NOT NULL,
    base_url            TEXT NOT NULL,
    category            VARCHAR(100) NOT NULL,
    source_type         VARCHAR(50) NOT NULL,
    city                VARCHAR(100),
    priority            VARCHAR(10) NOT NULL DEFAULT 'medium',
    tier                SMALLINT NOT NULL DEFAULT 2,
    extraction_schema   JSONB,
    crawl_config        JSONB,
    firecrawl_method    VARCHAR(20) NOT NULL DEFAULT 'extract',
    cron_expression     VARCHAR(100),
    next_run_at         TIMESTAMPTZ,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    health_score        DECIMAL(3,2) DEFAULT 1.00,
    total_runs          INT NOT NULL DEFAULT 0,
    successful_runs     INT NOT NULL DEFAULT 0,
    failed_runs         INT NOT NULL DEFAULT 0,
    last_run_at         TIMESTAMPTZ,
    last_success_at     TIMESTAMPTZ,
    last_failure_reason TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_sources_tenant_active ON webdata_sources(tenant_id, is_active);
CREATE INDEX idx_sources_next_run ON webdata_sources(next_run_at) WHERE is_active = true;
