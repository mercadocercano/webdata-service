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

func (wp *WorkerPool) runWorker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			wp.processNext(ctx, workerID)
		}
	}
}

func (wp *WorkerPool) processNext(ctx context.Context, workerID int) {
	job, err := wp.executeUC.ClaimJob(ctx)
	if err != nil {
		if _, ok := err.(exception.JobNotFoundError); ok {
			time.Sleep(workerPollInterval)
			return
		}
		fmt.Printf("[worker %d] claim error: %v\n", workerID, err)
		time.Sleep(workerPollInterval)
		return
	}

	if err := wp.executeUC.Execute(ctx, job); err != nil {
		fmt.Printf("[worker %d] execute error for job %s: %v\n", workerID, job.ID, err)
	}
}
