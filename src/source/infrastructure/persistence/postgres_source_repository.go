package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/entity"
	"github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
)

type PostgresSourceRepository struct {
	db *sql.DB
}

func NewPostgresSourceRepository(db *sql.DB) *PostgresSourceRepository {
	return &PostgresSourceRepository{db: db}
}

const sourceColumns = `
	id, tenant_id, name, base_url, category, source_type, city,
	priority, tier, extraction_schema, crawl_config, firecrawl_method,
	cron_expression, next_run_at, is_active, health_score,
	total_runs, successful_runs, failed_runs,
	last_run_at, last_success_at, last_failure_reason, notes,
	created_at, updated_at`

func (r *PostgresSourceRepository) Save(ctx context.Context, s *entity.Source) error {
	schemaJSON, _ := json.Marshal(s.ExtractionSchema.Raw())
	crawlJSON, _ := s.CrawlConfig.ToJSON()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO webdata_sources (
			id, tenant_id, name, base_url, category, source_type, city,
			priority, tier, extraction_schema, crawl_config, firecrawl_method,
			cron_expression, next_run_at, is_active, health_score,
			total_runs, successful_runs, failed_runs,
			last_run_at, last_success_at, last_failure_reason, notes,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25
		)`,
		s.ID, s.TenantID, s.Name, s.BaseURL, s.Category, s.SourceType, s.City,
		s.Priority.Value(), s.Tier.Value(), schemaJSON, crawlJSON, s.FirecrawlMethod,
		s.CronExpression, s.NextRunAt, s.IsActive, s.HealthScore,
		s.TotalRuns, s.SuccessfulRuns, s.FailedRuns,
		s.LastRunAt, s.LastSuccessAt, s.LastFailureReason, s.Notes,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting source: %w", err)
	}
	return nil
}

func (r *PostgresSourceRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.Source, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+sourceColumns+" FROM webdata_sources WHERE tenant_id=$1 AND id=$2",
		tenantID, id,
	)
	s, err := scanSource(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("source %s not found", id)
		}
		return nil, fmt.Errorf("scanning source: %w", err)
	}
	return s, nil
}

func (r *PostgresSourceRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.SourceFilter) ([]*entity.Source, int, error) {
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

	if filter.IsActive != nil {
		where += fmt.Sprintf(" AND is_active=$%d", n)
		args = append(args, *filter.IsActive)
		n++
	}
	if filter.Category != "" {
		where += fmt.Sprintf(" AND category=$%d", n)
		args = append(args, filter.Category)
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webdata_sources WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting sources: %w", err)
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+sourceColumns+" FROM webdata_sources WHERE "+where+
			fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n, n+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("querying sources: %w", err)
	}
	defer rows.Close()

	var sources []*entity.Source
	for rows.Next() {
		s, err := scanSource(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning source row: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, total, rows.Err()
}

func (r *PostgresSourceRepository) Update(ctx context.Context, s *entity.Source) error {
	s.UpdatedAt = time.Now()
	schemaJSON, _ := json.Marshal(s.ExtractionSchema.Raw())
	crawlJSON, _ := s.CrawlConfig.ToJSON()

	_, err := r.db.ExecContext(ctx, `
		UPDATE webdata_sources SET
			name=$3, base_url=$4, category=$5, source_type=$6, city=$7,
			priority=$8, tier=$9, extraction_schema=$10, crawl_config=$11,
			firecrawl_method=$12, cron_expression=$13, next_run_at=$14,
			is_active=$15, health_score=$16,
			total_runs=$17, successful_runs=$18, failed_runs=$19,
			last_run_at=$20, last_success_at=$21, last_failure_reason=$22,
			notes=$23, updated_at=$24
		WHERE tenant_id=$1 AND id=$2`,
		s.TenantID, s.ID,
		s.Name, s.BaseURL, s.Category, s.SourceType, s.City,
		s.Priority.Value(), s.Tier.Value(), schemaJSON, crawlJSON,
		s.FirecrawlMethod, s.CronExpression, s.NextRunAt,
		s.IsActive, s.HealthScore,
		s.TotalRuns, s.SuccessfulRuns, s.FailedRuns,
		s.LastRunAt, s.LastSuccessAt, s.LastFailureReason,
		s.Notes, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating source: %w", err)
	}
	return nil
}

func (r *PostgresSourceRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM webdata_sources WHERE tenant_id=$1 AND id=$2",
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("deleting source: %w", err)
	}
	return nil
}

func (r *PostgresSourceRepository) FindDueForScraping(ctx context.Context) ([]*entity.Source, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+sourceColumns+
			" FROM webdata_sources WHERE is_active=true AND next_run_at <= NOW() ORDER BY next_run_at ASC LIMIT 50",
	)
	if err != nil {
		return nil, fmt.Errorf("querying due sources: %w", err)
	}
	defer rows.Close()

	var sources []*entity.Source
	for rows.Next() {
		s, err := scanSource(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scanning source row: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

type scanFn func(dest ...interface{}) error

func scanSource(scan scanFn) (*entity.Source, error) {
	var s entity.Source
	var priorityStr string
	var tierVal int
	var schemaJSON, crawlJSON []byte

	err := scan(
		&s.ID, &s.TenantID, &s.Name, &s.BaseURL, &s.Category, &s.SourceType, &s.City,
		&priorityStr, &tierVal, &schemaJSON, &crawlJSON, &s.FirecrawlMethod,
		&s.CronExpression, &s.NextRunAt, &s.IsActive, &s.HealthScore,
		&s.TotalRuns, &s.SuccessfulRuns, &s.FailedRuns,
		&s.LastRunAt, &s.LastSuccessAt, &s.LastFailureReason, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	priority, err := value_object.NewSourcePriority(priorityStr)
	if err != nil {
		return nil, err
	}
	s.Priority = priority

	tier, err := value_object.NewSourceTier(tierVal)
	if err != nil {
		return nil, err
	}
	s.Tier = tier

	if len(schemaJSON) > 0 {
		var raw json.RawMessage
		if err := json.Unmarshal(schemaJSON, &raw); err == nil {
			schema, _ := value_object.NewExtractionSchema(raw)
			s.ExtractionSchema = schema
		}
	}

	return &s, nil
}
