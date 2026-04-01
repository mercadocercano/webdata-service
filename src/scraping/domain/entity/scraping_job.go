package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/value_object"
)

type ScrapingJob struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	SourceID        uuid.UUID
	Status          value_object.JobStatus
	TriggerType     value_object.TriggerType
	FirecrawlJobID  string
	ProductsFound   int
	ProductsSaved   int
	ErrorMessage    string
	StartedAt       *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
	RetryCount      int
	MaxRetries      int
}

func NewScrapingJob(tenantID, sourceID uuid.UUID, triggerType string) (*ScrapingJob, error) {
	tt, err := value_object.NewTriggerType(triggerType)
	if err != nil {
		return nil, err
	}

	return &ScrapingJob{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SourceID:    sourceID,
		Status:      value_object.JobStatusPending(),
		TriggerType: tt,
		CreatedAt:   time.Now(),
		MaxRetries:  3,
	}, nil
}

// NewJobStatusFromString is a convenience for test assertions.
func NewJobStatusFromString(v string) (value_object.JobStatus, error) {
	return value_object.NewJobStatus(v)
}

func (j *ScrapingJob) Start() error {
	if !j.Status.IsPending() {
		return fmt.Errorf("cannot start job in status %q, must be pending", j.Status.Value())
	}
	now := time.Now()
	j.Status = value_object.JobStatusRunning()
	j.StartedAt = &now
	return nil
}

func (j *ScrapingJob) Complete(found, saved int) error {
	if !j.Status.IsRunning() {
		return fmt.Errorf("cannot complete job in status %q, must be running", j.Status.Value())
	}
	now := time.Now()
	j.Status = value_object.JobStatusCompleted()
	j.ProductsFound = found
	j.ProductsSaved = saved
	j.CompletedAt = &now
	return nil
}

func (j *ScrapingJob) Fail(reason string) error {
	if !j.Status.IsRunning() {
		return fmt.Errorf("cannot fail job in status %q, must be running", j.Status.Value())
	}
	now := time.Now()
	j.Status = value_object.JobStatusFailed()
	j.ErrorMessage = reason
	j.CompletedAt = &now
	return nil
}

func (j *ScrapingJob) Cancel() error {
	if !j.Status.IsPending() {
		return fmt.Errorf("cannot cancel job in status %q, only pending jobs can be cancelled", j.Status.Value())
	}
	j.Status = value_object.JobStatusCancelled()
	return nil
}

func (j *ScrapingJob) CanRetry() bool {
	return j.RetryCount < j.MaxRetries
}
