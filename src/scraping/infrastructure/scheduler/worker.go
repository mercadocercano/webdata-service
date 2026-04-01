package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/mercadocercano/webdata-service/src/scraping/domain/exception"
	scrapingusecase "github.com/mercadocercano/webdata-service/src/scraping/application/usecase"
)

const workerPollInterval = 5 * time.Second

type WorkerPool struct {
	executeUC   *scrapingusecase.ExecuteScrapingUseCase
	workerCount int
}

func NewWorkerPool(executeUC *scrapingusecase.ExecuteScrapingUseCase, workerCount int) *WorkerPool {
	return &WorkerPool{executeUC: executeUC, workerCount: workerCount}
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
					fmt.Printf("[worker %d] claim error: %v\n", workerID, err)
				}
				continue
			}
			if err := wp.executeUC.Execute(ctx, job); err != nil {
				fmt.Printf("[worker %d] execute error for job %s: %v\n", workerID, job.ID, err)
			}
		}
	}
}
