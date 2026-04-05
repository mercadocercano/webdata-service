package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
)

// schedulingLockDuration es el tiempo mínimo entre que se crea un job y el scheduler
// puede volver a crear otro para la misma fuente. Evita el runaway de jobs duplicados.
const schedulingLockDuration = 10 * time.Minute

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
		maxPages := src.CrawlConfig.MaxPages
		if maxPages <= 0 {
			// No pagination: create a single job (page=0, legacy behavior).
			s.createJob(ctx, src, 0)
		} else {
			// Pagination: create one job per page (1..MaxPages).
			for page := 1; page <= maxPages; page++ {
				s.createJob(ctx, src, page)
			}
		}

		// Avanzar next_run_at para evitar que el scheduler cree jobs duplicados
		// antes de que el worker procese este job.
		s.lockSourceUntilProcessed(ctx, src)
	}
}

func (s *Scheduler) createJob(ctx context.Context, src *sourceentity.Source, page int) {
	var job *entity.ScrapingJob
	var err error

	if page > 0 {
		job, err = entity.NewPaginatedScrapingJob(src.TenantID, src.ID, "scheduled", page)
	} else {
		job, err = entity.NewScrapingJob(src.TenantID, src.ID, "scheduled")
	}
	if err != nil {
		fmt.Printf("[scheduler] error creating job for source %s page %d: %v\n", src.ID, page, err)
		return
	}

	if err := s.jobRepo.Save(ctx, job); err != nil {
		fmt.Printf("[scheduler] error saving job for source %s page %d: %v\n", src.ID, page, err)
	}
}

func (s *Scheduler) lockSourceUntilProcessed(ctx context.Context, src *sourceentity.Source) {
	nextRun := time.Now().Add(schedulingLockDuration)
	src.NextRunAt = &nextRun
	if err := s.sourceRepo.Update(ctx, src); err != nil {
		fmt.Printf("[scheduler] error locking next_run_at for source %s: %v\n", src.ID, err)
	}
}
