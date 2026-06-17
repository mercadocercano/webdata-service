package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/entity"
	scrapingport "github.com/mercadocercano/webdata-service/src/scraping/domain/port"
	sourceentity "github.com/mercadocercano/webdata-service/src/source/domain/entity"
	sourceport "github.com/mercadocercano/webdata-service/src/source/domain/port"
	"github.com/robfig/cron/v3"
)

// schedulingFallbackDuration es el tiempo de espera cuando el cron_expression
// es inválido o vacío. Evita re-scrapeo inmediato sin perder la fuente en el loop.
const schedulingFallbackDuration = 10 * time.Minute

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

		// Calcular el próximo next_run_at a partir del cron_expression.
		// Esto garantiza que la fuente no vuelva a estar "due" hasta el
		// horario real definido en el cron — evita el re-scrapeo a los 10 min.
		s.advanceNextRunAt(ctx, src)
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

// advanceNextRunAt calcula el próximo next_run_at usando el cron_expression de la
// fuente y lo persiste. Si el cron_expression es inválido o vacío, aplica un
// fallback de schedulingFallbackDuration para evitar que la fuente entre en loop.
// El disparo manual (POST /sources/{id}/trigger) no pasa por aquí: crea el job
// directamente sin tocar next_run_at.
func (s *Scheduler) advanceNextRunAt(ctx context.Context, src *sourceentity.Source) {
	nextRun := nextRunFromCron(src.CronExpression, time.Now())
	if err := s.sourceRepo.UpdateNextRunAt(ctx, src.ID, nextRun); err != nil {
		fmt.Printf("[scheduler] error updating next_run_at for source %s: %v\n", src.ID, err)
	}
}

// nextRunFromCron calcula el siguiente instante de ejecución para un cron_expression
// estándar (5 campos) a partir de `from`. Si el expression es inválido o vacío,
// devuelve from + schedulingFallbackDuration y loguea el error.
func nextRunFromCron(expr string, from time.Time) time.Time {
	if expr == "" {
		fmt.Printf("[scheduler] cron_expression vacío, aplicando fallback de %s\n", schedulingFallbackDuration)
		return from.Add(schedulingFallbackDuration)
	}

	schedule, err := cron.ParseStandard(expr)
	if err != nil {
		fmt.Printf("[scheduler] cron_expression inválido %q: %v — aplicando fallback de %s\n",
			expr, err, schedulingFallbackDuration)
		return from.Add(schedulingFallbackDuration)
	}

	return schedule.Next(from)
}
