-- Add page column to scraping_jobs for paginated scraping support.
-- Page 0 means no pagination (legacy behavior).
-- Page 1+ means the job scrapes a specific page number.
ALTER TABLE webdata_scraping_jobs ADD COLUMN page SMALLINT NOT NULL DEFAULT 0;
