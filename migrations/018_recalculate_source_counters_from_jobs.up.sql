-- Migration 018: Recalculate source counters from webdata_scraping_jobs (MER-169)
--
-- Race condition fix: scheduler and workers were doing full-row UPDATE on webdata_sources,
-- causing stale overwrites of counters. This migration recalculates accurate values
-- from the authoritative webdata_scraping_jobs table.

BEGIN;

UPDATE webdata_sources s SET
    total_runs = COALESCE(j.total, 0),
    successful_runs = COALESCE(j.successful, 0),
    failed_runs = COALESCE(j.failed, 0),
    health_score = CASE
        WHEN COALESCE(j.total, 0) = 0 THEN 1.0
        ELSE COALESCE(j.successful, 0)::float / j.total
    END
FROM (
    SELECT
        source_id,
        COUNT(*) FILTER (WHERE status IN ('completed', 'failed')) AS total,
        COUNT(*) FILTER (WHERE status = 'completed') AS successful,
        COUNT(*) FILTER (WHERE status = 'failed') AS failed
    FROM webdata_scraping_jobs
    GROUP BY source_id
) j
WHERE s.id = j.source_id;

COMMIT;
