package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

const pollInterval = 60 * time.Second

type Scheduler struct {
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapingport.ScrapingJobRepository
}

func NewScheduler(sourceRepo sourceport.SourceRepository, jobRepo scrapingport.ScrapingJobRepository) *Scheduler {
	return &Scheduler{sourceRepo: sourceRepo, jobRepo: jobRepo}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scheduleRuns(ctx)
		}
	}
}

func (s *Scheduler) scheduleRuns(ctx context.Context) {
	sources, err := s.sourceRepo.FindDueForScraping(ctx)
	if err != nil {
		fmt.Printf("[scheduler] error finding due sources: %v\n", err)
		return
	}

	for _, src := range sources {
		job, err := entity.NewScrapingJob(src.TenantID, src.ID, "scheduled")
		if err != nil {
			fmt.Printf("[scheduler] error creating job for source %s: %v\n", src.ID, err)
			continue
		}

		if err := s.jobRepo.Save(ctx, job); err != nil {
			fmt.Printf("[scheduler] error saving job for source %s: %v\n", src.ID, err)
		}
	}
}
