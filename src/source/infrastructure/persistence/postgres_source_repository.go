package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/shared/database"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/mercadocercano/webdata-service/src/source/domain/exception"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
)

type PostgresSourceRepository struct {
	db *sql.DB
}

func NewPostgresSourceRepository(db *sql.DB) *PostgresSourceRepository {
	return &PostgresSourceRepository{db: db}
}

func (r *PostgresSourceRepository) Save(ctx context.Context, s *entity.Source) error {
	query := `
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type, city,
			priority, tier, firecrawl_method, cron_expression, is_active,
			health_score, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`

	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.TenantID, s.Name, s.BaseURL, s.Category, s.SourceType, s.City,
		s.Priority.Value(), s.Tier.Value(), s.FirecrawlMethod, s.CronExpression,
		s.IsActive, s.HealthScore, s.Notes, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving source: %w", err)
	}
	return nil
}

func (r *PostgresSourceRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Source, error) {
	cols := `id, tenant_id, name, base_url, category, source_type, city,
		priority, tier, extraction_schema, crawl_config, firecrawl_method, cron_expression, next_run_at, is_active,
		health_score, total_runs, successful_runs, failed_runs,
		last_run_at, last_success_at, last_failure_reason, consecutive_failures, notes, created_at, updated_at`

	var row *sql.Row
	if tenantID == database.GlobalTenantID {
		query := `SELECT ` + cols + ` FROM webdata_sources WHERE id=$1 AND is_active=true`
		row = r.db.QueryRowContext(ctx, query, id)
	} else {
		query := `SELECT ` + cols + ` FROM webdata_sources WHERE tenant_id=$1 AND id=$2 AND is_active=true`
		row = r.db.QueryRowContext(ctx, query, tenantID, id)
	}
	return r.scanSource(row)
}

func (r *PostgresSourceRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.SourceFilter) ([]*entity.Source, int, error) {
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

	if filter.IsActive != nil {
		where += fmt.Sprintf(" AND is_active=$%d", argIdx)
		args = append(args, *filter.IsActive)
		argIdx++
	}
	if filter.Category != "" {
		where += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, filter.Category)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM webdata_sources " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting sources: %w", err)
	}

	query := `SELECT id, tenant_id, name, base_url, category, source_type, city,
		priority, tier, extraction_schema, crawl_config, firecrawl_method, cron_expression, next_run_at, is_active,
		health_score, total_runs, successful_runs, failed_runs,
		last_run_at, last_success_at, last_failure_reason, consecutive_failures, notes, created_at, updated_at
		FROM webdata_sources ` + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing sources: %w", err)
	}
	defer rows.Close()

	var sources []*entity.Source
	for rows.Next() {
		s, err := r.scanSourceRow(rows)
		if err != nil {
			return nil, 0, err
		}
		sources = append(sources, s)
	}
	return sources, total, rows.Err()
}

func (r *PostgresSourceRepository) Update(ctx context.Context, s *entity.Source) error {
	s.UpdatedAt = time.Now()
	query := `UPDATE webdata_sources SET
		name=$1, base_url=$2, category=$3, source_type=$4, city=$5,
		priority=$6, tier=$7, firecrawl_method=$8, cron_expression=$9,
		next_run_at=$10, is_active=$11, health_score=$12, total_runs=$13, successful_runs=$14,
		failed_runs=$15, last_run_at=$16, last_success_at=$17,
		last_failure_reason=$18, consecutive_failures=$19, notes=$20, updated_at=$21
		WHERE tenant_id=$22 AND id=$23`

	_, err := r.db.ExecContext(ctx, query,
		s.Name, s.BaseURL, s.Category, s.SourceType, s.City,
		s.Priority.Value(), s.Tier.Value(), s.FirecrawlMethod, s.CronExpression,
		s.NextRunAt, s.IsActive, s.HealthScore, s.TotalRuns, s.SuccessfulRuns,
		s.FailedRuns, s.LastRunAt, s.LastSuccessAt,
		s.LastFailureReason, s.ConsecutiveFailures, s.Notes, s.UpdatedAt,
		s.TenantID, s.ID,
	)
	if err != nil {
		return fmt.Errorf("updating source: %w", err)
	}
	return nil
}

func (r *PostgresSourceRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `UPDATE webdata_sources SET is_active=false, updated_at=$1 WHERE tenant_id=$2 AND id=$3`
	result, err := r.db.ExecContext(ctx, query, time.Now(), tenantID, id)
	if err != nil {
		return fmt.Errorf("deleting source: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return exception.SourceNotFoundError{ID: id.String()}
	}
	return nil
}

func (r *PostgresSourceRepository) FindDueForScraping(ctx context.Context) ([]*entity.Source, error) {
	query := `SELECT id, tenant_id, name, base_url, category, source_type, city,
		priority, tier, extraction_schema, crawl_config, firecrawl_method, cron_expression, next_run_at, is_active,
		health_score, total_runs, successful_runs, failed_runs,
		last_run_at, last_success_at, last_failure_reason, consecutive_failures, notes, created_at, updated_at
		FROM webdata_sources
		WHERE is_active=true AND cron_expression != ''
		AND (next_run_at IS NULL OR next_run_at <= NOW())`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("finding due sources: %w", err)
	}
	defer rows.Close()

	var sources []*entity.Source
	for rows.Next() {
		s, err := r.scanSourceRow(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *PostgresSourceRepository) scanSource(row *sql.Row) (*entity.Source, error) {
	var s entity.Source
	var priorityStr string
	var tierVal int
	var nextRunAt, lastRunAt, lastSuccessAt sql.NullTime
	var lastFailureReason, city, cronExpr, notes sql.NullString
	var extractionSchemaRaw, crawlConfigRaw []byte

	err := row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.BaseURL, &s.Category, &s.SourceType, &city,
		&priorityStr, &tierVal, &extractionSchemaRaw, &crawlConfigRaw, &s.FirecrawlMethod, &cronExpr, &nextRunAt, &s.IsActive,
		&s.HealthScore, &s.TotalRuns, &s.SuccessfulRuns, &s.FailedRuns,
		&lastRunAt, &lastSuccessAt, &lastFailureReason, &s.ConsecutiveFailures, &notes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, exception.SourceNotFoundError{}
	}
	if err != nil {
		return nil, fmt.Errorf("scanning source: %w", err)
	}

	return r.buildSource(&s, priorityStr, tierVal, city, cronExpr, notes, nextRunAt, lastRunAt, lastSuccessAt, lastFailureReason, extractionSchemaRaw, crawlConfigRaw)
}

func (r *PostgresSourceRepository) scanSourceRow(rows *sql.Rows) (*entity.Source, error) {
	var s entity.Source
	var priorityStr string
	var tierVal int
	var nextRunAt, lastRunAt, lastSuccessAt sql.NullTime
	var lastFailureReason, city, cronExpr, notes sql.NullString
	var extractionSchemaRaw, crawlConfigRaw []byte

	err := rows.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.BaseURL, &s.Category, &s.SourceType, &city,
		&priorityStr, &tierVal, &extractionSchemaRaw, &crawlConfigRaw, &s.FirecrawlMethod, &cronExpr, &nextRunAt, &s.IsActive,
		&s.HealthScore, &s.TotalRuns, &s.SuccessfulRuns, &s.FailedRuns,
		&lastRunAt, &lastSuccessAt, &lastFailureReason, &s.ConsecutiveFailures, &notes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning source row: %w", err)
	}

	return r.buildSource(&s, priorityStr, tierVal, city, cronExpr, notes, nextRunAt, lastRunAt, lastSuccessAt, lastFailureReason, extractionSchemaRaw, crawlConfigRaw)
}

func (r *PostgresSourceRepository) buildSource(
	s *entity.Source, priorityStr string, tierVal int,
	city, cronExpr, notes sql.NullString,
	nextRunAt, lastRunAt, lastSuccessAt sql.NullTime,
	lastFailureReason sql.NullString,
	extractionSchemaRaw, crawlConfigRaw []byte,
) (*entity.Source, error) {
	priority, err := value_object.NewSourcePriority(priorityStr)
	if err != nil {
		return nil, err
	}
	tier, err := value_object.NewSourceTier(tierVal)
	if err != nil {
		return nil, err
	}

	schema, err := value_object.NewExtractionSchema(extractionSchemaRaw)
	if err != nil {
		return nil, fmt.Errorf("building extraction schema: %w", err)
	}
	crawlCfg, err := value_object.CrawlConfigFromJSON(crawlConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("building crawl config: %w", err)
	}

	s.Priority = priority
	s.Tier = tier
	s.ExtractionSchema = schema
	s.CrawlConfig = crawlCfg
	if city.Valid { s.City = city.String }
	if cronExpr.Valid { s.CronExpression = cronExpr.String }
	if notes.Valid { s.Notes = notes.String }
	if nextRunAt.Valid { s.NextRunAt = &nextRunAt.Time }
	if lastRunAt.Valid { s.LastRunAt = &lastRunAt.Time }
	if lastSuccessAt.Valid { s.LastSuccessAt = &lastSuccessAt.Time }
	if lastFailureReason.Valid { s.LastFailureReason = lastFailureReason.String }

	return s, nil
}
