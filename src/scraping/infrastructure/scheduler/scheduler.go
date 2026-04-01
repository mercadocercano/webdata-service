package scheduler

import (
	"context"
	"fmt"
	"time"

	scrapeport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

const schedulerInterval = 60 * time.Second

type Scheduler struct {
	sourceRepo sourceport.SourceRepository
	jobRepo    scrapeport.ScrapingJobRepository
}

func NewScheduler(sourceRepo sourceport.SourceRepository, jobRepo scrapeport.ScrapingJobRepository) *Scheduler {
	return &Scheduler{sourceRepo: sourceRepo, jobRepo: jobRepo}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.enqueueJobs(ctx)
		}
	}
}

func (s *Scheduler) enqueueJobs(ctx context.Context) {
	sources, err := s.sourceRepo.FindDueForScraping(ctx)
	if err != nil {
		fmt.Printf("scheduler: error finding due sources: %v\n", err)
		return
	}

	for _, src := range sources {
		job, err := entity.NewScrapingJob(src.TenantID, src.ID, "scheduled")
		if err != nil {
			fmt.Printf("scheduler: error creating job for source %s: %v\n", src.ID, err)
			continue
		}
		if err := s.jobRepo.Save(ctx, job); err != nil {
			fmt.Printf("scheduler: error saving job for source %s: %v\n", src.ID, err)
			continue
		}
		fmt.Printf("scheduler: enqueued job %s for source %s\n", job.ID, src.ID)
	}
}
