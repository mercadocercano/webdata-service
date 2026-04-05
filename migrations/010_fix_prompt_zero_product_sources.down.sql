-- Migration 010 rollback: Remove prompts added to zero-product sources
-- Note: zombie job status changes are NOT reversed (jobs were genuinely stuck)

UPDATE webdata_sources SET prompt = NULL
WHERE name IN ('Coto', 'Farmacity', 'Diarco', 'Vital');
