CREATE TABLE webdata_products (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    source_id           UUID NOT NULL REFERENCES webdata_sources(id),
    job_id              UUID REFERENCES webdata_scraping_jobs(id),
    title               TEXT NOT NULL,
    price               DECIMAL(12,2),
    currency            VARCHAR(3) NOT NULL DEFAULT 'ARS',
    original_price      DECIMAL(12,2),
    url                 TEXT NOT NULL,
    image_url           TEXT,
    description         TEXT,
    brand               VARCHAR(255),
    category            VARCHAR(255),
    sku                 VARCHAR(100),
    ean                 VARCHAR(20),
    in_stock            BOOLEAN DEFAULT true,
    normalized_category VARCHAR(255),
    confidence_score    DECIMAL(3,2),
    content_hash        VARCHAR(64) NOT NULL,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    price_changed_at    TIMESTAMPTZ,
    raw_data            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_products_dedup ON webdata_products(tenant_id, source_id, content_hash);
CREATE INDEX idx_products_tenant_source ON webdata_products(tenant_id, source_id);
CREATE INDEX idx_products_category ON webdata_products(tenant_id, normalized_category);
CREATE INDEX idx_products_price ON webdata_products(tenant_id, price) WHERE price IS NOT NULL;
CREATE INDEX idx_products_last_seen ON webdata_products(last_seen_at);
