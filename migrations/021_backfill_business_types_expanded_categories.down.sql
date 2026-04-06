-- Remove business types assigned by migration 021
DELETE FROM webdata_product_business_types
WHERE product_id IN (
    SELECT p.id
    FROM webdata_products p
    JOIN webdata_sources s ON p.source_id = s.id AND p.tenant_id = s.tenant_id
    WHERE s.category IN (
        'almacen', 'lacteos', 'limpieza', 'bebidas', 'congelados', 'desayuno',
        'golosinas', 'aceites', 'panaderia', 'snacks', 'bazar', 'mascotas',
        'electrodomesticos', 'tecnologia', 'cuidado-personal', 'bebes',
        'calzado', 'construccion_ferreteria', 'sanitarios'
    )
);
