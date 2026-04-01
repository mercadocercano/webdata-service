CREATE TABLE webdata_price_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    product_id  UUID NOT NULL REFERENCES webdata_products(id) ON DELETE CASCADE,
    price       DECIMAL(12,2) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_price_history_product ON webdata_price_history(product_id, recorded_at DESC);
