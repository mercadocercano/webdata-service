-- Migration 029 rollback: Remove authoritative_category from webdata_sources

ALTER TABLE webdata_sources
    DROP COLUMN IF EXISTS authoritative_category;
