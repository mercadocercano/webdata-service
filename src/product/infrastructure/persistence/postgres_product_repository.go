package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type PostgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) *PostgresProductRepository {
	return &PostgresProductRepository{db: db}
}

const productColumns = `
	id, tenant_id, source_id, job_id, title, price, currency, original_price,
	url, image_url, description, brand, category, sku, ean, in_stock,
	normalized_category, confidence_score, content_hash,
	first_seen_at, last_seen_at, price_changed_at, raw_data,
	created_at, updated_at`

func (r *PostgresProductRepository) Upsert(ctx context.Context, p *entity.ScrapedProduct) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO webdata_products (
			id, tenant_id, source_id, job_id, title, price, currency, original_price,
			url, image_url, description, brand, category, sku, ean, in_stock,
			normalized_category, confidence_score, content_hash,
			first_seen_at, last_seen_at, price_changed_at, raw_data,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		ON CONFLICT (tenant_id, source_id, content_hash) DO UPDATE SET
			title=EXCLUDED.title, price=EXCLUDED.price, original_price=EXCLUDED.original_price,
			image_url=EXCLUDED.image_url, description=EXCLUDED.description,
			in_stock=EXCLUDED.in_stock, last_seen_at=EXCLUDED.last_seen_at,
			price_changed_at=EXCLUDED.price_changed_at, raw_data=EXCLUDED.raw_data,
			updated_at=EXCLUDED.updated_at`,
		p.ID, p.TenantID, p.SourceID, p.JobID, p.Title, p.Price, p.Currency, p.OriginalPrice,
		p.URL, p.ImageURL, p.Description, p.Brand, p.Category, p.SKU, p.EAN, p.InStock,
		p.NormalizedCategory, p.ConfidenceScore, p.ContentHash.String(),
		p.FirstSeenAt, p.LastSeenAt, p.PriceChangedAt, p.RawData,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("upserting product: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows == 1, nil
}

func (r *PostgresProductRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapedProduct, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+productColumns+" FROM webdata_products WHERE tenant_id=$1 AND id=$2",
		tenantID, id,
	)
	p, err := scanProduct(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("product %s not found", id)
		}
		return nil, fmt.Errorf("scanning product: %w", err)
	}
	return p, nil
}

func (r *PostgresProductRepository) FindByContentHash(ctx context.Context, tenantID, sourceID uuid.UUID, hash value_object.ContentHash) (*entity.ScrapedProduct, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+productColumns+" FROM webdata_products WHERE tenant_id=$1 AND source_id=$2 AND content_hash=$3",
		tenantID, sourceID, hash.String(),
	)
	p, err := scanProduct(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("product with hash %s not found", hash)
		}
		return nil, fmt.Errorf("scanning product by hash: %w", err)
	}
	return p, nil
}

func (r *PostgresProductRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.ProductFilter) ([]*entity.ScrapedProduct, int, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	where := "tenant_id=$1"
	args := []interface{}{tenantID}
	n := 2

	if filter.SourceID != nil {
		where += fmt.Sprintf(" AND source_id=$%d", n)
		args = append(args, *filter.SourceID)
		n++
	}
	if filter.Category != "" {
		where += fmt.Sprintf(" AND category=$%d", n)
		args = append(args, filter.Category)
		n++
	}
	if filter.Brand != "" {
		where += fmt.Sprintf(" AND brand=$%d", n)
		args = append(args, filter.Brand)
		n++
	}
	if filter.MinPrice != nil {
		where += fmt.Sprintf(" AND price>=$%d", n)
		args = append(args, *filter.MinPrice)
		n++
	}
	if filter.MaxPrice != nil {
		where += fmt.Sprintf(" AND price<=$%d", n)
		args = append(args, *filter.MaxPrice)
		n++
	}
	if filter.Query != "" {
		where += fmt.Sprintf(" AND title ILIKE $%d", n)
		args = append(args, "%"+filter.Query+"%")
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webdata_products WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting products: %w", err)
	}

	orderBy := "last_seen_at DESC"
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		orderBy = filter.SortBy + " " + order
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+productColumns+" FROM webdata_products WHERE "+where+
			fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, n, n+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("querying products: %w", err)
	}
	defer rows.Close()

	var products []*entity.ScrapedProduct
	for rows.Next() {
		p, err := scanProduct(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning product row: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *PostgresProductRepository) SavePriceRecord(ctx context.Context, record value_object.PriceRecord) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO webdata_price_history (tenant_id, product_id, price, recorded_at) VALUES ($1,$2,$3,$4)",
		record.TenantID, record.ProductID, record.Price, record.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("saving price record: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) FindPriceHistory(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.PriceRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT tenant_id, product_id, price, recorded_at FROM webdata_price_history WHERE tenant_id=$1 AND product_id=$2 ORDER BY recorded_at DESC",
		tenantID, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying price history: %w", err)
	}
	defer rows.Close()

	var records []value_object.PriceRecord
	for rows.Next() {
		var r value_object.PriceRecord
		if err := rows.Scan(&r.TenantID, &r.ProductID, &r.Price, &r.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning price record: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

type scanFn func(dest ...interface{}) error

func scanProduct(scan scanFn) (*entity.ScrapedProduct, error) {
	var p entity.ScrapedProduct
	var hashStr string

	err := scan(
		&p.ID, &p.TenantID, &p.SourceID, &p.JobID, &p.Title, &p.Price, &p.Currency, &p.OriginalPrice,
		&p.URL, &p.ImageURL, &p.Description, &p.Brand, &p.Category, &p.SKU, &p.EAN, &p.InStock,
		&p.NormalizedCategory, &p.ConfidenceScore, &hashStr,
		&p.FirstSeenAt, &p.LastSeenAt, &p.PriceChangedAt, &p.RawData,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.ContentHash = value_object.ContentHash(hashStr)
	return &p, nil
}
