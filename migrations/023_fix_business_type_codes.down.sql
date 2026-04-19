-- Rollback: restaurar códigos originales de webdata
UPDATE webdata_product_business_types
SET code = 'almacen_supermercado', name = 'Almacén / Supermercado'
WHERE code = 'almacen';

UPDATE webdata_product_business_types
SET code = 'almacen_mayorista', name = 'Almacén / Mayorista'
WHERE code = 'supermercado';

UPDATE webdata_product_business_types
SET code = 'indumentaria', name = 'Indumentaria'
WHERE code = 'ropa';

UPDATE webdata_product_business_types
SET name = 'Electrónica'
WHERE code = 'electronica';
