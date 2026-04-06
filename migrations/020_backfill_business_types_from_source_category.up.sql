-- Backfill business types for existing products based on source category.
-- Only assigns to products that have NO business types yet.

INSERT INTO webdata_product_business_types (product_id, business_type_code, business_type_name, tenant_id, created_at)
SELECT p.id, m.business_type_code, m.business_type_name, p.tenant_id, NOW()
FROM webdata_products p
JOIN webdata_sources s ON p.source_id = s.id AND p.tenant_id = s.tenant_id
JOIN (VALUES
    ('supermercado',           'almacen_supermercado', 'Almacén / Supermercado'),
    ('supermercado_mayorista', 'almacen_mayorista',    'Almacén / Mayorista'),
    ('electronica',           'electronica',           'Electrónica'),
    ('farmacia_perfumeria',   'farmacia',              'Farmacia'),
    ('ferreteria_construccion','ferreteria',            'Ferretería'),
    ('construccion_hogar',    'ferreteria',             'Ferretería'),
    ('indumentaria',          'indumentaria',           'Indumentaria')
) AS m(source_category, business_type_code, business_type_name)
ON s.category = m.source_category
WHERE NOT EXISTS (
    SELECT 1 FROM webdata_product_business_types bt WHERE bt.product_id = p.id
)
AND p.hidden_at IS NULL;
