CREATE TABLE IF NOT EXISTS product_image_resolutions (
    gtin VARCHAR(14) PRIMARY KEY,
    ean VARCHAR(14),
    product_id VARCHAR(36),
    source VARCHAR(20) NOT NULL,
    raw_url TEXT,
    cdn_url TEXT NOT NULL,
    cdn_key TEXT NOT NULL,
    enhancer VARCHAR(20) NOT NULL DEFAULT 'passthrough',
    cost_cents INTEGER NOT NULL DEFAULT 0,
    quality_score SMALLINT,
    resolved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_img_resolutions_quality ON product_image_resolutions(quality_score) WHERE quality_score < 3;
CREATE INDEX IF NOT EXISTS idx_img_resolutions_source ON product_image_resolutions(source);
