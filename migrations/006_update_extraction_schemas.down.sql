-- Migration 006 rollback: Revert extraction_schema to original values (without image_url, url, description)

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"unit":{"type":"string"},"sku":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.maxiconsumo.com';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"unit":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.diarco.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"originalPrice":{"type":"number"},"brand":{"type":"string"},"category":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.carrefour.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"installments":{"type":"string"},"brand":{"type":"string"},"model":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.electromisiones.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"laboratoire":{"type":"string"},"activeIngredient":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmacity.com';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"unit":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.vital.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"brand":{"type":"string"},"category":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.coto.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"brand":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.laanonima.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"installments":{"type":"string"},"brand":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.cetrogar.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"brand":{"type":"string"},"category":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.easy.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"brand":{"type":"string"},"sku":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.sodimac.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"brand":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmalife.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"installments":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.naldo.com.ar';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"installments":{"type":"string"},"brand":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.megatone.net';

UPDATE webdata_sources SET extraction_schema = '{"type":"object","properties":{"products":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"discount":{"type":"string"}}}}}}'::jsonb
WHERE base_url = 'https://www.farmar.com.ar';
