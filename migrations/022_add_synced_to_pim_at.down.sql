DROP INDEX IF EXISTS idx_products_pending_sync;
ALTER TABLE webdata_products DROP COLUMN IF EXISTS synced_to_pim_at;
