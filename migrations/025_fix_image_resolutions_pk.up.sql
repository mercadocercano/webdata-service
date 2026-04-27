-- Fix: gtin como PK era VARCHAR(14) pero el key puede ser nombre de producto.
-- Usamos product_id (UUID) como PK, gtin queda como columna nullable con unique index.

ALTER TABLE product_image_resolutions DROP CONSTRAINT IF EXISTS product_image_resolutions_pkey;

ALTER TABLE product_image_resolutions
    ALTER COLUMN gtin TYPE TEXT,
    ALTER COLUMN ean TYPE TEXT,
    ALTER COLUMN product_id TYPE TEXT;

ALTER TABLE product_image_resolutions ADD PRIMARY KEY (product_id);

-- gtin no es unique: distintos product_ids pueden mapear al mismo GTIN (variantes del mismo producto base)
CREATE INDEX IF NOT EXISTS idx_img_resolutions_gtin ON product_image_resolutions(gtin) WHERE gtin IS NOT NULL AND gtin != '';
