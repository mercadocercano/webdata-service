-- Migration 011: Fix source URLs based on CTO investigation (MER-125)
--
-- Findings:
-- - Farmacity: URL /coleccion/folleto no tiene productos. /medicamentos-venta-libre sí.
--   Confirmado via Firecrawl: 6 productos con precio e imagen en primer scrape.
-- - Coto: Sitio JSP/AJAX pesado. Firecrawl extract devuelve 0 productos en TODAS las URLs.
--   Requiere implementar soporte para método 'crawl' o scrape+parse. Desactivar por ahora.
-- - Diarco: No es ecommerce. Solo publica folletos PDF de ofertas semanales. No hay listados
--   de productos estructurados. Desactivar.
-- - Vital (Supermayorista Vital): No es ecommerce. Solo promos físicas y folletos.
--   No hay tarjetas de producto para extraer. Desactivar.

-- Farmacity: corregir URL a listado de productos que funciona con extract
UPDATE webdata_sources SET
    base_url = 'https://www.farmacity.com/medicamentos-venta-libre'
WHERE name = 'Farmacity';

-- Coto: desactivar hasta implementar soporte crawl (sitio JSP no compatible con extract)
UPDATE webdata_sources SET
    is_active = false
WHERE name = 'Coto';

-- Diarco: desactivar (no es ecommerce, solo folletos PDF)
UPDATE webdata_sources SET
    is_active = false
WHERE name = 'Diarco';

-- Vital: desactivar (no es ecommerce, solo promos físicas)
UPDATE webdata_sources SET
    is_active = false
WHERE name = 'Vital';
