-- Revierte: restaura NOT NULL en gtin y ean
-- ADVERTENCIA: fallará si hay filas con gtin/ean NULL
UPDATE product_image_resolutions SET gtin = '' WHERE gtin IS NULL;
UPDATE product_image_resolutions SET ean  = '' WHERE ean  IS NULL;

ALTER TABLE product_image_resolutions
    ALTER COLUMN gtin SET NOT NULL,
    ALTER COLUMN ean  SET NOT NULL;
