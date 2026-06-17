package scheduler

import (
	"context"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/exception"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
	webdataport "github.com/mercadocercano/webdata-service/src/webdata/domain/port"
)

const workerPollInterval = 5 * time.Second

type WorkerPool struct {
	executeUC   *scrapingusecase.ExecuteScrapingUseCase
	workerCount int
	logger      webdataport.WebdataEventLogger
}

func NewWorkerPool(executeUC *scrapingusecase.ExecuteScrapingUseCase, workerCount int) *WorkerPool {
	return &WorkerPool{executeUC: executeUC, workerCount: workerCount}
}

// WithLogger inyecta el logger canónico (ADR-001). Nil-safe.
func (wp *WorkerPool) WithLogger(logger webdataport.WebdataEventLogger) {
	wp.logger = logger
}

func (wp *WorkerPool) logEvt(e webdataport.WebdataEvent) {
	if wp.logger != nil {
		wp.logger.Log(e)
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workerCount; i++ {
		go wp.runWorker(ctx, i)
	}
	<-ctx.Done()
}

// runWorker uses a Ticker to poll for jobs at a fixed interval, ensuring
// the goroutine responds promptly to context cancellation (graceful drain).
func (wp *WorkerPool) runWorker(ctx context.Context, workerID int) {
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job, err := wp.executeUC.ClaimJob(ctx)
			if err != nil {
				if _, ok := err.(exception.JobNotFoundError); !ok {
					wp.logEvt(webdataport.WebdataEvent{
						Event:  "webdata.worker_claim_error",
						Reason: err.Error(),
					})
				}
				continue
			}
			if err := wp.executeUC.Execute(ctx, job); err != nil {
				wp.logEvt(webdataport.WebdataEvent{
					Event:    "webdata.worker_execute_error",
					JobID:    job.ID.String(),
					TenantID: job.TenantID.String(),
					SourceID: job.SourceID.String(),
					Reason:   err.Error(),
				})
			}
		}
	}
}
