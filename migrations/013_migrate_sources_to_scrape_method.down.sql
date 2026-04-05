-- Migration 013 rollback: restaurar firecrawl_method a 'extract' para las 3 fuentes

UPDATE webdata_sources
SET firecrawl_method = 'extract'
WHERE name IN ('Cetrogar', 'Electro Misiones', 'La Anónima');
