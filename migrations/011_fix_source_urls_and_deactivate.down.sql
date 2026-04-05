-- Migration 011 rollback: Restore original URLs and reactivate sources

UPDATE webdata_sources SET
    base_url = 'https://www.farmacity.com/coleccion/folleto'
WHERE name = 'Farmacity';

UPDATE webdata_sources SET
    is_active = true
WHERE name IN ('Coto', 'Diarco', 'Vital');
