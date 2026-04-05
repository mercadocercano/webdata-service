package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/shared/database"
	"github.com/mercadocercano/webdata-service/src/product/domain/entity"
	"github.com/mercadocercano/webdata-service/src/product/domain/exception"
	"github.com/mercadocercano/webdata-service/src/product/domain/port"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type PostgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) *PostgresProductRepository {
	return &PostgresProductRepository{db: db}
}

func (r *PostgresProductRepository) Upsert(ctx context.Context, p *entity.ScrapedProduct) (bool, error) {
	query := `
		INSERT INTO webdata_products (
			id, tenant_id, source_id, job_id, title, price, currency, original_price,
			url, image_url, description, brand, category, sku, ean, in_stock,
			normalized_category, confidence_score, content_hash, is_blocked, hidden_at,
			first_seen_at, last_seen_at, price_changed_at, raw_data, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		ON CONFLICT (tenant_id, source_id, content_hash) DO UPDATE SET
			last_seen_at = EXCLUDED.last_seen_at,
			price = EXCLUDED.price,
			price_changed_at = EXCLUDED.price_changed_at,
			image_url = EXCLUDED.image_url,
			category = EXCLUDED.category,
			brand = EXCLUDED.brand,
			description = EXCLUDED.description,
			url = EXCLUDED.url,
			in_stock = EXCLUDED.in_stock,
			updated_at = EXCLUDED.updated_at
		WHERE webdata_products.is_blocked = false
		  AND webdata_products.hidden_at IS NULL
		RETURNING (xmax = 0) AS inserted`

	// Use interface{} nil so lib/pq sends NULL for JSONB, not an empty byte slice.
	var rawDataJSON interface{}
	if len(p.RawData) > 0 {
		rawDataJSON = []byte(p.RawData)
	}

	var inserted bool
	err := r.db.QueryRowContext(ctx, query,
		p.ID, p.TenantID, p.SourceID, p.JobID, p.Title, p.Price, p.Currency, p.OriginalPrice,
		p.URL, p.ImageURL, p.Description, p.Brand, p.Category, p.SKU, p.EAN, p.InStock,
		p.NormalizedCategory, p.ConfidenceScore, p.ContentHash.String(),
		p.FirstSeenAt, p.LastSeenAt, p.PriceChangedAt, rawDataJSON, p.CreatedAt, p.UpdatedAt,
	).Scan(&inserted)

	if err != nil {
		return false, fmt.Errorf("upserting product: %w", err)
	}
	return inserted, nil
}

func (r *PostgresProductRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapedProduct, error) {
	query := `SELECT id, tenant_id, source_id, job_id, title, price, currency, original_price,
		url, image_url, description, brand, category, sku, ean, in_stock,
		normalized_category, confidence_score, content_hash,
		first_seen_at, last_seen_at, price_changed_at, raw_data, created_at, updated_at
		FROM webdata_products WHERE tenant_id=$1 AND id=$2`

	row := r.db.QueryRowContext(ctx, query, tenantID, id)
	return r.scanProduct(row)
}

func (r *PostgresProductRepository) FindByContentHash(ctx context.Context, tenantID, sourceID uuid.UUID, hash value_object.ContentHash) (*entity.ScrapedProduct, error) {
	query := `SELECT id, tenant_id, source_id, job_id, title, price, currency, original_price,
		url, image_url, description, brand, category, sku, ean, in_stock,
		normalized_category, confidence_score, content_hash,
		first_seen_at, last_seen_at, price_changed_at, raw_data, created_at, updated_at
		FROM webdata_products WHERE tenant_id=$1 AND source_id=$2 AND content_hash=$3`

	row := r.db.QueryRowContext(ctx, query, tenantID, sourceID, hash.String())
	return r.scanProduct(row)
}

func (r *PostgresProductRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.ProductFilter) ([]*entity.ScrapedProduct, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 9999 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where, args, argIdx := database.TenantWhereClause(tenantID)
	where += " AND hidden_at IS NULL"

	if filter.SourceID != nil {
		where += fmt.Sprintf(" AND source_id=$%d", argIdx)
		args = append(args, *filter.SourceID)
		argIdx++
	}
	if filter.Category != "" {
		where += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, filter.Category)
		argIdx++
	}
	if filter.Brand != "" {
		where += fmt.Sprintf(" AND brand=$%d", argIdx)
		args = append(args, filter.Brand)
		argIdx++
	}
	if filter.MinPrice != nil {
		where += fmt.Sprintf(" AND price >= $%d", argIdx)
		args = append(args, *filter.MinPrice)
		argIdx++
	}
	if filter.MaxPrice != nil {
		where += fmt.Sprintf(" AND price <= $%d", argIdx)
		args = append(args, *filter.MaxPrice)
		argIdx++
	}
	if filter.Query != "" {
		where += fmt.Sprintf(" AND (title ILIKE $%d OR brand ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Query+"%")
		argIdx++
	}
	if filter.BusinessTypeCode != "" {
		where += fmt.Sprintf(" AND id IN (SELECT product_id FROM webdata_product_business_types WHERE business_type_code = $%d AND tenant_id = $%d)", argIdx, argIdx+1)
		args = append(args, filter.BusinessTypeCode, tenantID)
		argIdx += 2
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webdata_products "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting products: %w", err)
	}

	sortBy := "last_seen_at"
	sortOrder := "DESC"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, source_id, job_id, title, price, currency, original_price,
		url, image_url, description, brand, category, sku, ean, in_stock,
		normalized_category, confidence_score, content_hash,
		first_seen_at, last_seen_at, price_changed_at, raw_data, created_at, updated_at
		FROM webdata_products %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, sortBy, sortOrder, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing products: %w", err)
	}
	defer rows.Close()

	var products []*entity.ScrapedProduct
	for rows.Next() {
		p, err := r.scanProductRow(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *PostgresProductRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE webdata_products SET hidden_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND hidden_at IS NULL`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("soft deleting product: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return exception.ProductNotFoundError{}
	}
	return nil
}

func (r *PostgresProductRepository) BulkSoftDelete(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := `UPDATE webdata_products SET hidden_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND hidden_at IS NULL AND id = ANY($2)`
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = id.String()
	}
	res, err := r.db.ExecContext(ctx, query, tenantID, fmt.Sprintf("{%s}", joinStrings(idStrs, ",")))
	if err != nil {
		return 0, fmt.Errorf("bulk soft deleting products: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (r *PostgresProductRepository) UpdateBlocked(ctx context.Context, tenantID, id uuid.UUID, blocked bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE webdata_products SET is_blocked = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND hidden_at IS NULL`,
		tenantID, id, blocked,
	)
	if err != nil {
		return fmt.Errorf("updating product blocked status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return exception.ProductNotFoundError{}
	}
	return nil
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}

func (r *PostgresProductRepository) SavePriceRecord(ctx context.Context, record value_object.PriceRecord) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO webdata_price_history (id, tenant_id, product_id, price, recorded_at) VALUES ($1,$2,$3,$4,$5)`,
		uuid.New(), record.TenantID, record.ProductID, record.Price, record.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("saving price record: %w", err)
	}
	return nil
}

func (r *PostgresProductRepository) FindPriceHistory(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.PriceRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT product_id, price, recorded_at FROM webdata_price_history
		WHERE product_id=$1 AND tenant_id=$2
		ORDER BY recorded_at DESC`,
		productID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding price history: %w", err)
	}
	defer rows.Close()

	var records []value_object.PriceRecord
	for rows.Next() {
		var rec value_object.PriceRecord
		rec.TenantID = tenantID
		if err := rows.Scan(&rec.ProductID, &rec.Price, &rec.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning price record: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *PostgresProductRepository) SaveBusinessTypes(ctx context.Context, tenantID, productID uuid.UUID, assignments []value_object.BusinessTypeAssignment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`DELETE FROM webdata_product_business_types WHERE product_id = $1 AND tenant_id = $2`,
		productID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("clearing existing business types: %w", err)
	}

	for _, a := range assignments {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO webdata_product_business_types (product_id, business_type_code, business_type_name, tenant_id, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			productID, a.BusinessTypeCode, a.BusinessTypeName, tenantID, a.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("inserting business type assignment %q: %w", a.BusinessTypeCode, err)
		}
	}

	return tx.Commit()
}

func (r *PostgresProductRepository) RemoveBusinessType(ctx context.Context, tenantID, productID uuid.UUID, code string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM webdata_product_business_types WHERE product_id = $1 AND tenant_id = $2 AND business_type_code = $3`,
		productID, tenantID, code,
	)
	if err != nil {
		return fmt.Errorf("removing business type: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("business type %q not assigned to product", code)
	}
	return nil
}

func (r *PostgresProductRepository) FindBusinessTypesForProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]value_object.BusinessTypeAssignment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT business_type_code, business_type_name, created_at
		 FROM webdata_product_business_types
		 WHERE product_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC`,
		productID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding business types for product: %w", err)
	}
	defer rows.Close()

	var assignments []value_object.BusinessTypeAssignment
	for rows.Next() {
		var a value_object.BusinessTypeAssignment
		if err := rows.Scan(&a.BusinessTypeCode, &a.BusinessTypeName, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning business type assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

func (r *PostgresProductRepository) scanProduct(row *sql.Row) (*entity.ScrapedProduct, error) {
	var p entity.ScrapedProduct
	var jobID sql.NullString
	var price, originalPrice, confidenceScore sql.NullFloat64
	var imageURL, description, brand, category, sku, ean, normalizedCategory sql.NullString
	var hiddenAt, priceChangedAt sql.NullTime
	var rawData []byte
	var contentHashStr string

	err := row.Scan(
		&p.ID, &p.TenantID, &p.SourceID, &jobID, &p.Title, &price, &p.Currency, &originalPrice,
		&p.URL, &imageURL, &description, &brand, &category, &sku, &ean, &p.InStock,
		&normalizedCategory, &confidenceScore, &contentHashStr, &p.IsBlocked, &hiddenAt,
		&p.FirstSeenAt, &p.LastSeenAt, &priceChangedAt, &rawData, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, exception.ProductNotFoundError{}
	}
	if err != nil {
		return nil, fmt.Errorf("scanning product: %w", err)
	}

	if hiddenAt.Valid {
		p.HiddenAt = &hiddenAt.Time
	}

	return buildProduct(&p, jobID, price, originalPrice, confidenceScore,
		imageURL, description, brand, category, sku, ean, normalizedCategory,
		priceChangedAt, rawData, contentHashStr)
}

func (r *PostgresProductRepository) scanProductRow(rows *sql.Rows) (*entity.ScrapedProduct, error) {
	var p entity.ScrapedProduct
	var jobID sql.NullString
	var price, originalPrice, confidenceScore sql.NullFloat64
	var imageURL, description, brand, category, sku, ean, normalizedCategory sql.NullString
	var hiddenAt, priceChangedAt sql.NullTime
	var rawData []byte
	var contentHashStr string

	err := rows.Scan(
		&p.ID, &p.TenantID, &p.SourceID, &jobID, &p.Title, &price, &p.Currency, &originalPrice,
		&p.URL, &imageURL, &description, &brand, &category, &sku, &ean, &p.InStock,
		&normalizedCategory, &confidenceScore, &contentHashStr, &p.IsBlocked, &hiddenAt,
		&p.FirstSeenAt, &p.LastSeenAt, &priceChangedAt, &rawData, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning product row: %w", err)
	}

	if hiddenAt.Valid {
		p.HiddenAt = &hiddenAt.Time
	}

	return buildProduct(&p, jobID, price, originalPrice, confidenceScore,
		imageURL, description, brand, category, sku, ean, normalizedCategory,
		priceChangedAt, rawData, contentHashStr)
}

func buildProduct(
	p *entity.ScrapedProduct,
	jobID sql.NullString,
	price, originalPrice, confidenceScore sql.NullFloat64,
	imageURL, description, brand, category, sku, ean, normalizedCategory sql.NullString,
	priceChangedAt sql.NullTime,
	rawData []byte,
	contentHashStr string,
) (*entity.ScrapedProduct, error) {
	if jobID.Valid {
		id, err := uuid.Parse(jobID.String)
		if err == nil {
			p.JobID = &id
		}
	}
	if price.Valid { p.Price = &price.Float64 }
	if originalPrice.Valid { p.OriginalPrice = &originalPrice.Float64 }
	if confidenceScore.Valid { p.ConfidenceScore = &confidenceScore.Float64 }
	if imageURL.Valid { p.ImageURL = imageURL.String }
	if description.Valid { p.Description = description.String }
	if brand.Valid { p.Brand = brand.String }
	if category.Valid { p.Category = category.String }
	if sku.Valid { p.SKU = sku.String }
	if ean.Valid { p.EAN = ean.String }
	if normalizedCategory.Valid { p.NormalizedCategory = normalizedCategory.String }
	if priceChangedAt.Valid { p.PriceChangedAt = &priceChangedAt.Time }
	if len(rawData) > 0 { p.RawData = json.RawMessage(rawData) }

	p.ContentHash = value_object.ContentHash(contentHashStr)
	return p, nil
}
