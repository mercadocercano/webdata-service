-- Fix: alinear códigos de business_type con PIM service
-- webdata usaba códigos consolidados que no existen en PIM

UPDATE webdata_product_business_types
SET code = 'almacen', name = 'Almacén de Barrio'
WHERE code = 'almacen_supermercado';

UPDATE webdata_product_business_types
SET code = 'supermercado', name = 'Supermercado'
WHERE code = 'almacen_mayorista';

UPDATE webdata_product_business_types
SET code = 'ropa', name = 'Tienda de Ropa'
WHERE code = 'indumentaria';

UPDATE webdata_product_business_types
SET name = 'Casa de Electrodomésticos'
WHERE code = 'electronica';
