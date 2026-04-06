-- Rollback: remove all business types that were inserted by the backfill migration.
-- This deletes ALL entries since the table was empty before the backfill.
DELETE FROM webdata_product_business_types
WHERE created_at >= (
    SELECT MAX(created_at) FROM webdata_product_business_types
) - INTERVAL '1 minute';
