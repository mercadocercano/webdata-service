-- Junction table for M:N relationship between webdata_products and business_types (cross-database)
-- business_types live in pim_db, so we store code + name (denormalized) instead of UUID FK

CREATE TABLE webdata_product_business_types (
    product_id          UUID NOT NULL REFERENCES webdata_products(id) ON DELETE CASCADE,
    business_type_code  VARCHAR(50) NOT NULL,
    business_type_name  VARCHAR(255) NOT NULL DEFAULT '',
    tenant_id           UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (product_id, business_type_code)
);

CREATE INDEX idx_wpbt_tenant ON webdata_product_business_types(tenant_id);
CREATE INDEX idx_wpbt_business_type_code ON webdata_product_business_types(business_type_code);
CREATE INDEX idx_wpbt_tenant_code ON webdata_product_business_types(tenant_id, business_type_code);
