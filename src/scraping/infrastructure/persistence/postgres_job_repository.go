package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/exception"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/value_object"
)

type PostgresJobRepository struct {
	db *sql.DB
}

func NewPostgresJobRepository(db *sql.DB) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) Save(ctx context.Context, j *entity.ScrapingJob) error {
	query := `INSERT INTO webdata_scraping_jobs
		(id, tenant_id, source_id, status, trigger_type, retry_count, max_retries, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	_, err := r.db.ExecContext(ctx, query,
		j.ID, j.TenantID, j.SourceID, j.Status.Value(), j.TriggerType.Value(),
		j.RetryCount, j.MaxRetries, j.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving job: %w", err)
	}
	return nil
}

func (r *PostgresJobRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*entity.ScrapingJob, error) {
	query := `SELECT id, tenant_id, source_id, status, trigger_type, firecrawl_job_id,
		products_found, products_saved, error_message, started_at, completed_at,
		created_at, retry_count, max_retries
		FROM webdata_scraping_jobs WHERE tenant_id=$1 AND id=$2`

	row := r.db.QueryRowContext(ctx, query, tenantID, id)
	return r.scanJob(row)
}

func (r *PostgresJobRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter port.JobFilter) ([]*entity.ScrapingJob, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 9999 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where := "WHERE tenant_id=$1"
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.Status != "" {
		where += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.SourceID != nil {
		where += fmt.Sprintf(" AND source_id=$%d", argIdx)
		args = append(args, *filter.SourceID)
		argIdx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webdata_scraping_jobs "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting jobs: %w", err)
	}

	query := `SELECT id, tenant_id, source_id, status, trigger_type, firecrawl_job_id,
		products_found, products_saved, error_message, started_at, completed_at,
		created_at, retry_count, max_retries
		FROM webdata_scraping_jobs ` + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*entity.ScrapingJob
	for rows.Next() {
		j, err := r.scanJobRow(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, rows.Err()
}

func (r *PostgresJobRepository) Update(ctx context.Context, j *entity.ScrapingJob) error {
	query := `UPDATE webdata_scraping_jobs SET
		status=$1, firecrawl_job_id=$2, products_found=$3, products_saved=$4,
		error_message=$5, started_at=$6, completed_at=$7, retry_count=$8
		WHERE tenant_id=$9 AND id=$10`

	_, err := r.db.ExecContext(ctx, query,
		j.Status.Value(), j.FirecrawlJobID, j.ProductsFound, j.ProductsSaved,
		j.ErrorMessage, j.StartedAt, j.CompletedAt, j.RetryCount,
		j.TenantID, j.ID,
	)
	if err != nil {
		return fmt.Errorf("updating job: %w", err)
	}
	return nil
}

func (r *PostgresJobRepository) ClaimPendingJob(ctx context.Context) (*entity.ScrapingJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	query := `SELECT id, tenant_id, source_id, status, trigger_type, firecrawl_job_id,
		products_found, products_saved, error_message, started_at, completed_at,
		created_at, retry_count, max_retries
		FROM webdata_scraping_jobs
		WHERE status='pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	row := tx.QueryRowContext(ctx, query)
	job, err := r.scanJob(row)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`UPDATE webdata_scraping_jobs SET status='running', started_at=$1 WHERE id=$2`,
		now, job.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("claiming job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing claim: %w", err)
	}

	job.Status = value_object.JobStatusRunning()
	job.StartedAt = &now
	return job, nil
}

func (r *PostgresJobRepository) scanJob(row *sql.Row) (*entity.ScrapingJob, error) {
	var j entity.ScrapingJob
	var statusStr, triggerStr string
	var firecrawlJobID, errorMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := row.Scan(
		&j.ID, &j.TenantID, &j.SourceID, &statusStr, &triggerStr, &firecrawlJobID,
		&j.ProductsFound, &j.ProductsSaved, &errorMsg, &startedAt, &completedAt,
		&j.CreatedAt, &j.RetryCount, &j.MaxRetries,
	)
	if err == sql.ErrNoRows {
		return nil, exception.JobNotFoundError{}
	}
	if err != nil {
		return nil, fmt.Errorf("scanning job: %w", err)
	}

	return r.buildJob(&j, statusStr, triggerStr, firecrawlJobID, errorMsg, startedAt, completedAt)
}

func (r *PostgresJobRepository) scanJobRow(rows *sql.Rows) (*entity.ScrapingJob, error) {
	var j entity.ScrapingJob
	var statusStr, triggerStr string
	var firecrawlJobID, errorMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := rows.Scan(
		&j.ID, &j.TenantID, &j.SourceID, &statusStr, &triggerStr, &firecrawlJobID,
		&j.ProductsFound, &j.ProductsSaved, &errorMsg, &startedAt, &completedAt,
		&j.CreatedAt, &j.RetryCount, &j.MaxRetries,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning job row: %w", err)
	}

	return r.buildJob(&j, statusStr, triggerStr, firecrawlJobID, errorMsg, startedAt, completedAt)
}

func (r *PostgresJobRepository) buildJob(
	j *entity.ScrapingJob,
	statusStr, triggerStr string,
	firecrawlJobID, errorMsg sql.NullString,
	startedAt, completedAt sql.NullTime,
) (*entity.ScrapingJob, error) {
	status, err := value_object.NewJobStatus(statusStr)
	if err != nil {
		return nil, err
	}
	trigger, err := value_object.NewTriggerType(triggerStr)
	if err != nil {
		return nil, err
	}

	j.Status = status
	j.TriggerType = trigger
	if firecrawlJobID.Valid { j.FirecrawlJobID = firecrawlJobID.String }
	if errorMsg.Valid { j.ErrorMessage = errorMsg.String }
	if startedAt.Valid { j.StartedAt = &startedAt.Time }
	if completedAt.Valid { j.CompletedAt = &completedAt.Time }

	return j, nil
}
