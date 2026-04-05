-- Migration 017: Cleanup fuentes duplicadas e inactivas sin valor (MER-168)
--
-- Acciones:
-- 1. Para cada par de fuentes duplicadas (mismo nombre base + tenant_id):
--    mantener la que tiene más total_runs, eliminar la otra.
--    Reasignar productos huérfanos a la fuente retenida.
-- 2. Eliminar fuentes inactivas sin productos asociados.
-- 3. Blaisten: eliminar la fuente con URL rota (.com), mantener la correcta (.com.ar).
--
-- Pares duplicados: Carrefour Almuerzo, Carrefour Bebidas, Carrefour Congelados,
-- Carrefour Desayuno, Carrefour Electrodomesticos, Carrefour Lacteos,
-- Carrefour Limpieza, Dexter.

BEGIN;

-- ============================================================
-- 1. DUPLICADOS: reasignar productos y eliminar la fuente con menos runs
-- ============================================================

-- Para cada par duplicado (mismo name, mismo tenant_id), la fuente "winner"
-- es la que tiene mayor total_runs (desempate por successful_runs, luego id).
-- La "loser" se elimina después de reasignar sus productos.

-- 1a. Delete products from loser that already exist in winner (same content_hash)
--     to avoid unique constraint violation on idx_products_dedup(tenant_id, source_id, content_hash)
WITH duplicate_pairs AS (
    SELECT
        s.id,
        s.name,
        s.tenant_id,
        s.total_runs,
        s.successful_runs,
        ROW_NUMBER() OVER (
            PARTITION BY s.name, s.tenant_id
            ORDER BY s.total_runs DESC, s.successful_runs DESC, s.created_at ASC
        ) AS rn
    FROM webdata_sources s
    WHERE s.name IN (
        'Carrefour Almuerzo', 'Carrefour Bebidas', 'Carrefour Congelados',
        'Carrefour Desayuno', 'Carrefour Electrodomesticos', 'Carrefour Lacteos',
        'Carrefour Limpieza', 'Dexter'
    )
),
winners AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 1
),
losers AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 2
),
conflicting_products AS (
    SELECT lp.id
    FROM webdata_products lp
    JOIN losers l ON lp.source_id = l.id
    JOIN winners w ON w.name = l.name AND w.tenant_id = l.tenant_id
    JOIN webdata_products wp ON wp.source_id = w.id
        AND wp.tenant_id = lp.tenant_id
        AND wp.content_hash = lp.content_hash
)
DELETE FROM webdata_products WHERE id IN (SELECT id FROM conflicting_products);

-- 1b. Reasignar productos restantes de la perdedora a la ganadora
WITH duplicate_pairs AS (
    SELECT
        s.id,
        s.name,
        s.tenant_id,
        s.total_runs,
        s.successful_runs,
        ROW_NUMBER() OVER (
            PARTITION BY s.name, s.tenant_id
            ORDER BY s.total_runs DESC, s.successful_runs DESC, s.created_at ASC
        ) AS rn
    FROM webdata_sources s
    WHERE s.name IN (
        'Carrefour Almuerzo', 'Carrefour Bebidas', 'Carrefour Congelados',
        'Carrefour Desayuno', 'Carrefour Electrodomesticos', 'Carrefour Lacteos',
        'Carrefour Limpieza', 'Dexter'
    )
),
winners AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 1
),
losers AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 2
)
UPDATE webdata_products p
SET source_id = w.id
FROM losers l
JOIN winners w ON w.name = l.name AND w.tenant_id = l.tenant_id
WHERE p.source_id = l.id;

-- 1c. Reasignar scraping_jobs de la perdedora a la ganadora (para consistencia histórica)
WITH duplicate_pairs AS (
    SELECT
        s.id,
        s.name,
        s.tenant_id,
        ROW_NUMBER() OVER (
            PARTITION BY s.name, s.tenant_id
            ORDER BY s.total_runs DESC, s.successful_runs DESC, s.created_at ASC
        ) AS rn
    FROM webdata_sources s
    WHERE s.name IN (
        'Carrefour Almuerzo', 'Carrefour Bebidas', 'Carrefour Congelados',
        'Carrefour Desayuno', 'Carrefour Electrodomesticos', 'Carrefour Lacteos',
        'Carrefour Limpieza', 'Dexter'
    )
),
winners AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 1
),
losers AS (
    SELECT id, name, tenant_id FROM duplicate_pairs WHERE rn = 2
)
UPDATE webdata_scraping_jobs j
SET source_id = w.id
FROM losers l
JOIN winners w ON w.name = l.name AND w.tenant_id = l.tenant_id
WHERE j.source_id = l.id;

-- 1d. Eliminar las fuentes duplicadas perdedoras
WITH duplicate_pairs AS (
    SELECT
        s.id,
        s.name,
        s.tenant_id,
        ROW_NUMBER() OVER (
            PARTITION BY s.name, s.tenant_id
            ORDER BY s.total_runs DESC, s.successful_runs DESC, s.created_at ASC
        ) AS rn
    FROM webdata_sources s
    WHERE s.name IN (
        'Carrefour Almuerzo', 'Carrefour Bebidas', 'Carrefour Congelados',
        'Carrefour Desayuno', 'Carrefour Electrodomesticos', 'Carrefour Lacteos',
        'Carrefour Limpieza', 'Dexter'
    )
)
DELETE FROM webdata_sources
WHERE id IN (SELECT id FROM duplicate_pairs WHERE rn = 2);

-- ============================================================
-- 2. BLAISTEN: eliminar fuente con URL rota (.com), mantener .com.ar
-- ============================================================

-- Delete conflicting products (same content_hash already exists in keeper)
DELETE FROM webdata_products
WHERE id IN (
    SELECT bp.id
    FROM webdata_products bp
    JOIN webdata_sources broken ON bp.source_id = broken.id
    JOIN webdata_sources keeper ON keeper.base_url LIKE '%blaisten.com.ar/sanitarios%'
        AND broken.tenant_id = keeper.tenant_id
    JOIN webdata_products kp ON kp.source_id = keeper.id
        AND kp.tenant_id = bp.tenant_id
        AND kp.content_hash = bp.content_hash
    WHERE broken.base_url LIKE '%www.blaisten.com/sanitarios%'
      AND broken.base_url NOT LIKE '%blaisten.com.ar%'
);

-- Reasignar remaining productos
UPDATE webdata_products p
SET source_id = keeper.id
FROM webdata_sources broken,
     webdata_sources keeper
WHERE broken.base_url LIKE '%www.blaisten.com/sanitarios%'
  AND broken.base_url NOT LIKE '%blaisten.com.ar%'
  AND keeper.base_url LIKE '%blaisten.com.ar/sanitarios%'
  AND broken.tenant_id = keeper.tenant_id
  AND p.source_id = broken.id;

-- Reasignar jobs
UPDATE webdata_scraping_jobs j
SET source_id = keeper.id
FROM webdata_sources broken,
     webdata_sources keeper
WHERE broken.base_url LIKE '%www.blaisten.com/sanitarios%'
  AND broken.base_url NOT LIKE '%blaisten.com.ar%'
  AND keeper.base_url LIKE '%blaisten.com.ar/sanitarios%'
  AND broken.tenant_id = keeper.tenant_id
  AND j.source_id = broken.id;

-- Eliminar la fuente rota
DELETE FROM webdata_sources
WHERE base_url LIKE '%www.blaisten.com/sanitarios%'
  AND base_url NOT LIKE '%blaisten.com.ar%';

-- ============================================================
-- 3. INACTIVAS SIN PRODUCTOS: eliminar fuentes que no aportan valor
-- ============================================================

-- Eliminar jobs de fuentes inactivas sin productos (para evitar FK violations)
DELETE FROM webdata_scraping_jobs
WHERE source_id IN (
    SELECT s.id
    FROM webdata_sources s
    LEFT JOIN webdata_products p ON p.source_id = s.id
    WHERE s.is_active = false
    GROUP BY s.id
    HAVING COUNT(p.id) = 0
);

-- Eliminar las fuentes inactivas sin productos
DELETE FROM webdata_sources
WHERE id IN (
    SELECT s.id
    FROM webdata_sources s
    LEFT JOIN webdata_products p ON p.source_id = s.id
    WHERE s.is_active = false
    GROUP BY s.id
    HAVING COUNT(p.id) = 0
);

COMMIT;
