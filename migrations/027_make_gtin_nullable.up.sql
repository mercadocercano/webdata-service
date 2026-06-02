-- gtin ya no es PK (migración 025), pero sigue teniendo NOT NULL del esquema original.
-- Lo hacemos nullable para permitir registros sin GTIN (ej: reject manual de imagen).
ALTER TABLE product_image_resolutions
    ALTER COLUMN gtin DROP NOT NULL,
    ALTER COLUMN ean  DROP NOT NULL;
