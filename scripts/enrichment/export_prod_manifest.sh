#!/usr/bin/env bash
# export_prod_manifest.sh — genera el manifiesto de cambios de imágenes para migrar a PROD.
#
# Cada corrida de enrichment escribe en webdata_db.product_image_resolutions una fila por
# producto resuelto (product_id, source, cdn_url, quality_score, cost_cents, resolved_at).
# El ÚNICO cambio de datos que hay que llevar a prod es global_products.image_url → cdn_url.
# Las imágenes ya viven en el bucket Spaces compartido (= ya están en el CDN de prod);
# esto migra solo las FILAS de DB.
#
# Uso:
#   ./export_prod_manifest.sh "<since_timestamp>" "<rubro_label>"
# Ej:
#   ./export_prod_manifest.sh "2026-06-16 10:00:00" carniceria
#
# Capturá <since_timestamp> JUSTO ANTES de lanzar el batch del rubro:
#   START=$(docker exec lab-postgres psql -U postgres -d webdata_db -tAc "SELECT now()")
#
# Salidas (en ./out/):
#   enrichment-<rubro>-<fecha>.sql  → UPDATEs idempotentes para replicar en prod
#   enrichment-<rubro>-<fecha>.csv  → reporte auditable (producto, fuente, calidad, costo)

set -euo pipefail

SINCE="${1:?Falta <since_timestamp>, ej: '2026-06-16 10:00:00'}"
RUBRO="${2:?Falta <rubro_label>, ej: carniceria}"
PG="${PG_CONTAINER:-lab-postgres}"
DB="${WEBDATA_DB:-webdata_db}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="$(cd "$(dirname "$0")" && pwd)/out"
mkdir -p "$OUT_DIR"
SQL_FILE="$OUT_DIR/enrichment-${RUBRO}-${STAMP}.sql"
CSV_FILE="$OUT_DIR/enrichment-${RUBRO}-${STAMP}.csv"

# --- Manifiesto SQL: UPDATE idempotente por product_id ------------------------
# Solo filas con cdn_url real de Spaces (no fallbacks locales /tmp).
docker exec -i "$PG" psql -U postgres -d "$DB" -tA -F$'\t' -c "
  SELECT product_id, cdn_url, source, coalesce(quality_score,0)
  FROM product_image_resolutions
  WHERE resolved_at >= '${SINCE}'
    AND cdn_url ILIKE '%digitaloceanspaces.com%'
  ORDER BY resolved_at
" | awk -F'\t' -v rubro="$RUBRO" -v since="$SINCE" '
BEGIN {
  print "-- Manifiesto de migración a PROD — rubro: " rubro;
  print "-- Cambios de enrichment desde: " since;
  print "-- Aplicar contra pim_db de PROD. Idempotente: solo pisa image_url no-CDN.";
  print "BEGIN;";
}
{
  gsub(/'\''/, "'\'''\''", $2);  # escapar comillas simples en la URL
  printf "UPDATE global_products SET image_url='\''%s'\'', updated_at=now() WHERE id='\''%s'\'' AND image_url IS DISTINCT FROM '\''%s'\''; -- src=%s q=%s\n", $2, $1, $2, $3, $4;
  n++;
}
END {
  print "COMMIT;";
  print "-- Total UPDATEs: " n+0;
}
' > "$SQL_FILE"

# --- Reporte CSV auditable -----------------------------------------------------
docker exec -i "$PG" psql -U postgres -d "$DB" -c "\copy (
  SELECT product_id, gtin, ean, source, enhancer, quality_score, cost_cents, cdn_url, resolved_at
  FROM product_image_resolutions
  WHERE resolved_at >= '${SINCE}'
  ORDER BY resolved_at
) TO STDOUT WITH CSV HEADER" > "$CSV_FILE"

ROWS=$(grep -c '^UPDATE' "$SQL_FILE" || true)
COST=$(docker exec -i "$PG" psql -U postgres -d "$DB" -tAc "
  SELECT coalesce(sum(cost_cents),0) FROM product_image_resolutions WHERE resolved_at >= '${SINCE}'")

echo "Manifiesto generado para rubro '${RUBRO}':"
echo "  SQL : $SQL_FILE  (${ROWS} UPDATEs)"
echo "  CSV : $CSV_FILE"
echo "  Costo acumulado del batch: ${COST} centavos USD"
echo ""
echo "Para migrar a PROD (cuando se decida):"
echo "  cat '$SQL_FILE' | <psql apuntando a pim_db de prod>"
