-- Migration 010: Add prompts to zero-product sources + clean zombie jobs
--
-- Problema A: 8 fuentes con successful_runs > 0 pero 0 productos porque no tienen prompt
-- Fix: Agregar prompts descriptivos a las 4 fuentes prioritarias (Coto, Farmacity, Diarco, Vital)
--
-- Problema B: Jobs zombie en estado 'running' por más de 2 horas
-- Fix: Marcarlos como 'failed' con mensaje descriptivo

-- Coto (Cotodigital): supermercado nacional, catálogo general de alimentos/bebidas/limpieza
UPDATE webdata_sources SET
    prompt = 'Extrae todos los productos del catálogo del supermercado. Para image_url, usá el atributo src o data-src del tag <img> dentro de cada tarjeta de producto. Las URLs de imagen deben empezar con http y terminar en .jpg, .png, .webp o similar. Buscá nombre, precio y categoría de cada producto. Ignorá logos, banners y elementos de navegación.',
    extraction_schema = '{
      "type": "object",
      "properties": {
        "products": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name":        { "type": "string", "description": "Nombre completo del producto" },
              "price":       { "type": "number", "description": "Precio de venta actual en ARS" },
              "image_url":   { "type": "string", "description": "URL completa de la imagen principal del producto. Extraer del atributo src o data-src del tag <img> dentro de la tarjeta del producto. Debe empezar con http. Excluir logos, banners e imágenes genéricas." },
              "url":         { "type": "string", "description": "URL de la página del producto" },
              "brand":       { "type": "string", "description": "Marca del producto" },
              "category":    { "type": "string", "description": "Categoría del producto" },
              "description": { "type": "string", "description": "Descripción breve del producto" }
            }
          }
        }
      }
    }'::jsonb
WHERE name = 'Coto';

-- Farmacity: farmacia nacional, medicamentos, cosméticos, cuidado personal
UPDATE webdata_sources SET
    prompt = 'Extrae los productos de salud y farmacia de este listado. Para image_url, encontrá la imagen principal de cada tarjeta de producto. Buscá el atributo src o data-src del tag <img>. Las URLs deben apuntar a fotos reales del producto (.jpg, .png, .webp). Excluí logos del sitio, banners y elementos decorativos.',
    extraction_schema = '{
      "type": "object",
      "properties": {
        "products": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name":             { "type": "string", "description": "Nombre del producto farmacéutico o de salud" },
              "price":            { "type": "number", "description": "Precio en ARS" },
              "image_url":        { "type": "string", "description": "URL completa de la imagen del producto. Buscar atributo src o data-src del <img> en la tarjeta. Debe ser una URL de imagen real (.jpg, .png, .webp), no logos ni placeholders." },
              "url":              { "type": "string", "description": "URL de la página del producto" },
              "brand":            { "type": "string", "description": "Marca o laboratorio" },
              "category":         { "type": "string", "description": "Categoría (medicamentos, cosméticos, cuidado personal, etc.)" },
              "description":      { "type": "string", "description": "Descripción del producto" },
              "laboratoire":      { "type": "string", "description": "Laboratorio fabricante" },
              "activeIngredient": { "type": "string", "description": "Principio activo o ingrediente principal" }
            }
          }
        }
      }
    }'::jsonb
WHERE name = 'Farmacity';

-- Diarco: supermercado mayorista, alimentos/bebidas en volumen
UPDATE webdata_sources SET
    prompt = 'Extrae los productos del catálogo mayorista de alimentos y bebidas. Para image_url, usá el atributo src o data-src del tag <img> dentro de cada tarjeta de producto. Las URLs de imagen deben empezar con http y terminar en .jpg, .png, .webp. Registrá nombre, precio por bulto/unidad y categoría. Ignorá logos, banners y navegación.',
    extraction_schema = '{
      "type": "object",
      "properties": {
        "products": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name":        { "type": "string", "description": "Nombre completo del producto" },
              "price":       { "type": "number", "description": "Precio de venta en ARS (mayorista o por unidad)" },
              "image_url":   { "type": "string", "description": "URL completa de la imagen del producto. Extraer del atributo src o data-src del <img> en la tarjeta. Debe empezar con http. No logos ni banners." },
              "url":         { "type": "string", "description": "URL de la página del producto" },
              "brand":       { "type": "string", "description": "Marca del producto" },
              "category":    { "type": "string", "description": "Categoría del producto" },
              "description": { "type": "string", "description": "Descripción del producto" },
              "unit":        { "type": "string", "description": "Unidad de medida o presentación (kg, lt, un, bulto, etc.)" }
            }
          }
        }
      }
    }'::jsonb
WHERE name = 'Diarco';

-- Vital: farmacia/droguería regional, medicamentos y productos de salud con ofertas
UPDATE webdata_sources SET
    prompt = 'Extrae los productos en oferta de esta farmacia online. Para image_url, encontrá la imagen de cada producto dentro de su tarjeta. Buscá el atributo src o data-src en los tags <img>. También revisá data-lazy-src o srcset. Las URLs deben apuntar a fotos reales del producto. Excluí logos, banners e imágenes decorativas.',
    extraction_schema = '{
      "type": "object",
      "properties": {
        "products": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name":        { "type": "string", "description": "Nombre del producto de farmacia o droguería" },
              "price":       { "type": "number", "description": "Precio en ARS" },
              "image_url":   { "type": "string", "description": "URL completa de la imagen del producto. Revisar atributos src, data-src, data-lazy-src o srcset del <img> en la tarjeta del producto. Priorizar imagen de mayor resolución. No incluir logos ni imágenes decorativas." },
              "url":         { "type": "string", "description": "URL de la página del producto" },
              "brand":       { "type": "string", "description": "Marca o laboratorio" },
              "category":    { "type": "string", "description": "Categoría del producto" },
              "description": { "type": "string", "description": "Descripción del producto" },
              "unit":        { "type": "string", "description": "Unidad de medida o presentación" }
            }
          }
        }
      }
    }'::jsonb
WHERE name = 'Vital';

-- Limpiar zombie jobs: jobs en estado 'running' por más de 2 horas
UPDATE webdata_scraping_jobs
SET status = 'failed',
    error_message = 'Job zombie: marcado como failed por cleanup automático (timeout excedido > 2 horas)',
    completed_at = NOW()
WHERE status = 'running'
  AND started_at < NOW() - INTERVAL '2 hours';
