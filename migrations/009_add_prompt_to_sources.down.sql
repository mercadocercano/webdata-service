-- Migration 009 rollback: Remove prompt column from webdata_sources

ALTER TABLE webdata_sources DROP COLUMN IF EXISTS prompt;
