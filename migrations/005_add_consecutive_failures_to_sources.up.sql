ALTER TABLE webdata_sources
    ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0;
