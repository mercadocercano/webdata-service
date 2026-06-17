-- Migration 028: Add excluded_brands column to webdata_sources
-- Allows per-source brand filtering during product ingestion.
-- Brands in this list are discarded before persisting to webdata_products (case-insensitive trim match).

ALTER TABLE webdata_sources
    ADD COLUMN IF NOT EXISTS excluded_brands TEXT[] NOT NULL DEFAULT '{}';
