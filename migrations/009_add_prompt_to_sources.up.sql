-- Migration 009: Add prompt column to webdata_sources
-- Used by Extract/Scrape methods to pass contextual instructions to Firecrawl LLM extraction

ALTER TABLE webdata_sources ADD COLUMN IF NOT EXISTS prompt TEXT DEFAULT '';
