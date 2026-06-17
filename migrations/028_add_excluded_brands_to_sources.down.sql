-- Migration 028 rollback: Remove excluded_brands column from webdata_sources

ALTER TABLE webdata_sources
    DROP COLUMN IF EXISTS excluded_brands;
