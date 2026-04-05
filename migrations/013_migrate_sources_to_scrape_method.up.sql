-- Migration 013: Cambiar firecrawl_method a 'scrape' para fuentes que fallan con /v1/extract (MER-141)
--
-- Diagnóstico (MER-138): Firecrawl /v1/extract (async LLM) devuelve 0 productos para estas 3 fuentes.
-- El endpoint /v1/scrape con formats:["json"] + jsonOptions SÍ retorna productos estructurados:
--   - Cetrogar: 5+ productos confirmados
--   - Electro Misiones: 187 productos confirmados
--   - La Anónima: 5+ productos confirmados
--
-- Con esta migration las 3 fuentes usan ScrapeJSON (sync) en lugar de Extract (async LLM).

UPDATE webdata_sources
SET
    firecrawl_method      = 'scrape',
    consecutive_failures  = 0,
    is_active             = true,
    health_score          = 1.00
WHERE name IN ('Cetrogar', 'Electro Misiones', 'La Anónima');
