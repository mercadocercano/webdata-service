-- Migration 029: Add authoritative_category to webdata_sources (E26)
-- Para fuentes DEDICADAS de un solo rubro (Easy, Puppis, Blaisten), el mapeo de la
-- fuente (category → business_type) es la verdad. Con este flag, el upsert SALTEA el
-- resolver por-producto (keyword-based) y usa solo el mapeo de la fuente, evitando
-- ruido como "shampoo de perro → peluqueria" en una tienda de mascotas.
-- Fuentes multi-rubro (supers E22) lo dejan en FALSE → siguen usando el resolver per-producto.

ALTER TABLE webdata_sources
    ADD COLUMN IF NOT EXISTS authoritative_category BOOLEAN NOT NULL DEFAULT FALSE;
