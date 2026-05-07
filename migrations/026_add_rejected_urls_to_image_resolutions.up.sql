ALTER TABLE product_image_resolutions
    ADD COLUMN IF NOT EXISTS rejected_urls TEXT[] NOT NULL DEFAULT '{}';
