package entity

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
)

const (
	circuitBreakerThreshold = 5
	backoffBaseMinutes      = 5.0
	backoffMaxHours         = 24.0
)

type Source struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	Name             string
	BaseURL          string
	Category         string
	SourceType       string
	City             string
	Priority         value_object.SourcePriority
	Tier             value_object.SourceTier
	ExtractionSchema value_object.ExtractionSchema
	CrawlConfig      value_object.CrawlConfig
	FirecrawlMethod  string
	CronExpression   string
	NextRunAt        *time.Time
	IsActive         bool
	HealthScore      float64
	TotalRuns        int
	SuccessfulRuns   int
	FailedRuns       int
	LastRunAt        *time.Time
	LastSuccessAt      *time.Time
	LastFailureReason  string
	ConsecutiveFailures int
	Notes              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateSourceParams struct {
	TenantID         uuid.UUID
	Name             string
	BaseURL          string
	Category         string
	SourceType       string
	City             string
	Priority         string
	Tier             int
	FirecrawlMethod  string
	CronExpression   string
	Notes            string
	ExtractionSchema []byte
	CrawlConfigRaw   []byte
}

func NewSource(p CreateSourceParams) (*Source, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if p.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if p.Category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if p.SourceType == "" {
		return nil, fmt.Errorf("source_type is required")
	}

	priority, err := value_object.NewSourcePriority(p.Priority)
	if err != nil {
		return nil, err
	}

	tier, err := value_object.NewSourceTier(p.Tier)
	if err != nil {
		return nil, err
	}

	method := p.FirecrawlMethod
	if method == "" {
		method = "extract"
	}

	now := time.Now()
	return &Source{
		ID:              uuid.New(),
		TenantID:        p.TenantID,
		Name:            p.Name,
		BaseURL:         p.BaseURL,
		Category:        p.Category,
		SourceType:      p.SourceType,
		City:            p.City,
		Priority:        priority,
		Tier:            tier,
		FirecrawlMethod: method,
		CronExpression:  p.CronExpression,
		Notes:           p.Notes,
		IsActive:        true,
		HealthScore:     1.00,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *Source) RecordSuccess() {
	now := time.Now()
	s.TotalRuns++
	s.SuccessfulRuns++
	s.LastRunAt = &now
	s.LastSuccessAt = &now
	s.LastFailureReason = ""
	s.ConsecutiveFailures = 0
	s.recalculateHealthScore()
	s.UpdatedAt = now
}

func (s *Source) RecordFailure(reason string) {
	now := time.Now()
	s.TotalRuns++
	s.FailedRuns++
	s.LastRunAt = &now
	s.LastFailureReason = reason
	s.ConsecutiveFailures++
	s.recalculateHealthScore()

	if s.ConsecutiveFailures >= circuitBreakerThreshold {
		fmt.Printf("[circuit-breaker] source %s (%s) desactivada tras %d fallos consecutivos: %s\n",
			s.ID, s.Name, s.ConsecutiveFailures, reason)
		s.Deactivate()
	} else {
		backoffMinutes := backoffBaseMinutes * math.Pow(2, float64(s.ConsecutiveFailures-1))
		if backoffMinutes > backoffMaxHours*60 {
			backoffMinutes = backoffMaxHours * 60
		}
		nextRun := now.Add(time.Duration(backoffMinutes) * time.Minute)
		s.NextRunAt = &nextRun
		fmt.Printf("[backoff] source %s (%s) backoff de %.0f min (fallo #%d): %s\n",
			s.ID, s.Name, backoffMinutes, s.ConsecutiveFailures, reason)
	}

	s.UpdatedAt = now
}

func (s *Source) recalculateHealthScore() {
	if s.TotalRuns == 0 {
		s.HealthScore = 1.00
		return
	}
	s.HealthScore = float64(s.SuccessfulRuns) / float64(s.TotalRuns)
}

func (s *Source) Deactivate() {
	s.IsActive = false
	s.UpdatedAt = time.Now()
}
