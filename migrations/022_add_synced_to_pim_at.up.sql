ALTER TABLE webdata_products ADD COLUMN synced_to_pim_at TIMESTAMPTZ;

CREATE INDEX idx_products_pending_sync
    ON webdata_products (created_at)
    WHERE synced_to_pim_at IS NULL
      AND hidden_at IS NULL
      AND price IS NOT NULL
      AND image_url IS NOT NULL
      AND category IS NOT NULL;
