package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/entity"
	"github.com/mercadocercano/webdata-service/src/enrichment/domain/port"
)

type PostgresEnrichmentRepository struct {
	db *sql.DB
}

func NewPostgresEnrichmentRepository(db *sql.DB) *PostgresEnrichmentRepository {
	return &PostgresEnrichmentRepository{db: db}
}

func (r *PostgresEnrichmentRepository) IsResolved(ctx context.Context, productID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM product_image_resolutions WHERE product_id = $1 AND raw_url != ''",
		productID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking resolution: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresEnrichmentRepository) Save(ctx context.Context, res *entity.ImageResolution) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_image_resolutions
			(gtin, ean, product_id, source, raw_url, cdn_url, cdn_key, enhancer, cost_cents, quality_score, resolved_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (product_id) DO UPDATE SET
			gtin         = EXCLUDED.gtin,
			ean          = EXCLUDED.ean,
			source       = EXCLUDED.source,
			raw_url      = EXCLUDED.raw_url,
			cdn_url      = EXCLUDED.cdn_url,
			cdn_key      = EXCLUDED.cdn_key,
			enhancer     = EXCLUDED.enhancer,
			cost_cents   = EXCLUDED.cost_cents,
			quality_score = EXCLUDED.quality_score,
			updated_at   = NOW()
	`,
		res.GTIN, res.EAN, res.ProductID, string(res.Source),
		res.RawURL, res.CDNURL, res.CDNKey, string(res.Enhancer),
		res.CostCents, res.QualityScore, now, now,
	)
	if err != nil {
		return fmt.Errorf("saving resolution: %w", err)
	}
	return nil
}

func (r *PostgresEnrichmentRepository) RejectImage(ctx context.Context, productID string, imageURL string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_image_resolutions
			(product_id, source, raw_url, cdn_url, cdn_key, enhancer, rejected_urls, resolved_at, updated_at)
		VALUES ($1, 'manual', '', '', '', 'passthrough', ARRAY[$2::TEXT], NOW(), NOW())
		ON CONFLICT (product_id) DO UPDATE SET
			rejected_urls = CASE
				WHEN $2 = ANY(product_image_resolutions.rejected_urls)
				THEN product_image_resolutions.rejected_urls
				ELSE array_append(product_image_resolutions.rejected_urls, $2)
			END,
			raw_url    = '',
			updated_at = NOW()
	`, productID, imageURL)
	if err != nil {
		return fmt.Errorf("rejecting image for product %s: %w", productID, err)
	}
	return nil
}

func (r *PostgresEnrichmentRepository) GetRejectedURLs(ctx context.Context, productID string) ([]string, error) {
	var urls []string
	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(rejected_urls, '{}') FROM product_image_resolutions WHERE product_id = $1",
		productID,
	).Scan(pq.Array(&urls))
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting rejected URLs for product %s: %w", productID, err)
	}
	return urls, nil
}

func (r *PostgresEnrichmentRepository) GetCurrentRawURL(ctx context.Context, productID string) (string, error) {
	var rawURL string
	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(raw_url, '') FROM product_image_resolutions WHERE product_id = $1",
		productID,
	).Scan(&rawURL)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting current raw URL for product %s: %w", productID, err)
	}
	return rawURL, nil
}

func (r *PostgresEnrichmentRepository) GetStatus(ctx context.Context) (*port.EnrichmentStatus, error) {
	status := &port.EnrichmentStatus{
		BySource: make(map[string]int),
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(cost_cents), 0),
			COUNT(*) FILTER (WHERE quality_score IS NOT NULL AND quality_score < 3)
		FROM product_image_resolutions
	`)
	if err := row.Scan(&status.TotalResolved, &status.TotalCostCents, &status.LowQualityCount); err != nil {
		return nil, fmt.Errorf("getting totals: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT source, COUNT(*) FROM product_image_resolutions GROUP BY source
	`)
	if err != nil {
		return nil, fmt.Errorf("getting by-source stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var src string
		var count int
		if err := rows.Scan(&src, &count); err != nil {
			return nil, fmt.Errorf("scanning source row: %w", err)
		}
		status.BySource[src] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating source rows: %w", err)
	}

	return status, nil
}
