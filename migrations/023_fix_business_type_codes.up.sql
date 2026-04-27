-- Fix: alinear códigos de business_type con PIM service
-- webdata usaba códigos consolidados que no existen en PIM

UPDATE webdata_product_business_types
SET business_type_code = 'almacen', business_type_name = 'Almacén de Barrio'
WHERE business_type_code = 'almacen_supermercado';

UPDATE webdata_product_business_types
SET business_type_code = 'supermercado', business_type_name = 'Supermercado'
WHERE business_type_code = 'almacen_mayorista';

UPDATE webdata_product_business_types
SET business_type_code = 'ropa', business_type_name = 'Tienda de Ropa'
WHERE business_type_code = 'indumentaria';

UPDATE webdata_product_business_types
SET business_type_name = 'Casa de Electrodomésticos'
WHERE business_type_code = 'electronica';
