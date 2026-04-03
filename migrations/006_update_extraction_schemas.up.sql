-- Migration 006: Enrich extraction_schema for all 15 NEA sources
-- Adds image_url, url, brand, category, description to every schema
-- Preserves existing vertical-specific fields per source

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"},"sku":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.maxiconsumo.com';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.diarco.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"originalPrice":{"type":"number"}}}}}}'::jsonb
WHERE base_url = 'https://www.carrefour.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"},"model":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.electromisiones.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"laboratoire":{"type":"string"},"activeIngredient":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmacity.com';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"unit":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.vital.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.coto.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.laanonima.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.cetrogar.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.easy.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"sku":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.sodimac.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmalife.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.naldo.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"installments":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.megatone.net';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"image_url":{"type":"string"},"url":{"type":"string"},"brand":{"type":"string"},"category":{"type":"string"},"description":{"type":"string"},"discount":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmar.com.ar';
