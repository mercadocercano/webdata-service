-- Revierte migración 025: restaura gtin como PK VARCHAR(14)
-- ADVERTENCIA: destructivo si hay product_ids con gtin duplicado o más de 14 chars

DROP INDEX IF EXISTS idx_img_resolutions_gtin;

ALTER TABLE product_image_resolutions DROP CONSTRAINT IF EXISTS product_image_resolutions_pkey;

ALTER TABLE product_image_resolutions
    ALTER COLUMN gtin TYPE VARCHAR(14),
    ALTER COLUMN ean TYPE VARCHAR(14),
    ALTER COLUMN product_id TYPE VARCHAR(50);

ALTER TABLE product_image_resolutions ADD PRIMARY KEY (gtin);
