-- Backfill business types for products whose source categories were not covered
-- by migration 020. Adds mappings for subcategories (lacteos, limpieza, etc.)

INSERT INTO webdata_product_business_types (product_id, business_type_code, business_type_name, tenant_id, created_at)
SELECT p.id, m.business_type_code, m.business_type_name, p.tenant_id, NOW()
FROM webdata_products p
JOIN webdata_sources s ON p.source_id = s.id AND p.tenant_id = s.tenant_id
JOIN (VALUES
    ('almacen',                'almacen_supermercado', 'Almacén / Supermercado'),
    ('lacteos',               'almacen_supermercado', 'Almacén / Supermercado'),
    ('limpieza',              'almacen_supermercado', 'Almacén / Supermercado'),
    ('bebidas',               'almacen_supermercado', 'Almacén / Supermercado'),
    ('congelados',            'almacen_supermercado', 'Almacén / Supermercado'),
    ('desayuno',              'almacen_supermercado', 'Almacén / Supermercado'),
    ('golosinas',             'almacen_supermercado', 'Almacén / Supermercado'),
    ('aceites',               'almacen_supermercado', 'Almacén / Supermercado'),
    ('panaderia',             'almacen_supermercado', 'Almacén / Supermercado'),
    ('snacks',                'almacen_supermercado', 'Almacén / Supermercado'),
    ('bazar',                 'almacen_supermercado', 'Almacén / Supermercado'),
    ('mascotas',              'almacen_supermercado', 'Almacén / Supermercado'),
    ('electrodomesticos',     'electronica',          'Electrónica'),
    ('tecnologia',            'electronica',          'Electrónica'),
    ('cuidado-personal',      'farmacia',             'Farmacia'),
    ('bebes',                 'farmacia',             'Farmacia'),
    ('calzado',               'indumentaria',         'Indumentaria'),
    ('construccion_ferreteria','ferreteria',           'Ferretería'),
    ('sanitarios',            'ferreteria',            'Ferretería')
) AS m(source_category, business_type_code, business_type_name)
ON s.category = m.source_category
WHERE NOT EXISTS (
    SELECT 1 FROM webdata_product_business_types bt WHERE bt.product_id = p.id
)
AND p.hidden_at IS NULL;
