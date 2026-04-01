package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/value_object"
)

type PostgresJobRepository struct {
	db *sql.DB
}

func NewPostgresJobRepository(db *sql.DB) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

const jobColumns = `
	id, tenant_id, source_id, status, trigger_type,
	firecrawl_job_id, products_found, products_saved, error_message,
	started_at, completed_at, created_at, retry_count, max_retries`

func (r *PostgresJobRepository) Save(ctx context.Context, j *entity.ScrapingJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO webdata_scraping_jobs (
			id, tenant_id, source_id, status, trigger_type,
			firecrawl_job_id, products_found, products_saved, error_message,
			started_at, completed_at, created_at, retry_count, max_retries
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		j.ID, j.TenantID, j.SourceID, j.Status.Value(), j.TriggerType.Value(),
		j.FirecrawlJobID, j.ProductsFound, j.ProductsSaved, j.ErrorMessage,
		j.StartedAt, j.CompletedAt, j.CreatedAt, j.RetryCount, j.MaxRetries,
	)
	if err != nil {
		return fmt.Errorf("inserting job: %w", err)
	}
	return nil
}

func (r *PostgresJobRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapingJob, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+jobColumns+" FROM webdata_scraping_jobs WHERE tenant_id=$1 AND id=$2",
		tenantID, id,
	)
	j, err := scanJob(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("job %s not found", id)
		}
		return nil, fmt.Errorf("scanning job: %w", err)
	}
	return j, nil
}

func (r *PostgresJobRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.JobFilter) ([]*entity.ScrapingJob, int, error) {
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
	if filter.Status != "" {
		where += fmt.Sprintf(" AND status=$%d", n)
		args = append(args, filter.Status)
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webdata_scraping_jobs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting jobs: %w", err)
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+jobColumns+" FROM webdata_scraping_jobs WHERE "+where+
			fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n, n+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("querying jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*entity.ScrapingJob
	for rows.Next() {
		j, err := scanJob(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning job row: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, total, rows.Err()
}

func (r *PostgresJobRepository) Update(ctx context.Context, j *entity.ScrapingJob) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE webdata_scraping_jobs SET
			status=$3, firecrawl_job_id=$4, products_found=$5, products_saved=$6,
			error_message=$7, started_at=$8, completed_at=$9, retry_count=$10
		WHERE tenant_id=$1 AND id=$2`,
		j.TenantID, j.ID,
		j.Status.Value(), j.FirecrawlJobID, j.ProductsFound, j.ProductsSaved,
		j.ErrorMessage, j.StartedAt, j.CompletedAt, j.RetryCount,
	)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	return nil
}

// ClaimPendingJob uses FOR UPDATE SKIP LOCKED to safely claim a pending job in a worker pool.
func (r *PostgresJobRepository) ClaimPendingJob(ctx context.Context) (*entity.ScrapingJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	row := tx.QueryRowContext(ctx,
		"SELECT "+jobColumns+
			" FROM webdata_scraping_jobs WHERE status='pending' ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED",
	)
	j, err := scanJob(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no pending jobs")
		}
		return nil, fmt.Errorf("claiming job: %w", err)
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx,
		"UPDATE webdata_scraping_jobs SET status='running', started_at=$1 WHERE id=$2",
		now, j.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("updating claimed job status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing claim: %w", err)
	}

	js, _ := value_object.NewJobStatus("running")
	j.Status = js
	j.StartedAt = &now
	return j, nil
}

type scanFn func(dest ...interface{}) error

func scanJob(scan scanFn) (*entity.ScrapingJob, error) {
	var j entity.ScrapingJob
	var statusStr, triggerStr string

	err := scan(
		&j.ID, &j.TenantID, &j.SourceID, &statusStr, &triggerStr,
		&j.FirecrawlJobID, &j.ProductsFound, &j.ProductsSaved, &j.ErrorMessage,
		&j.StartedAt, &j.CompletedAt, &j.CreatedAt, &j.RetryCount, &j.MaxRetries,
	)
	if err != nil {
		return nil, err
	}

	status, err := value_object.NewJobStatus(statusStr)
	if err != nil {
		return nil, err
	}
	j.Status = status

	trigger, err := value_object.NewTriggerType(triggerStr)
	if err != nil {
		return nil, err
	}
	j.TriggerType = trigger

	return &j, nil
}
