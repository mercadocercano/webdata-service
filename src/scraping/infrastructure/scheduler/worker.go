package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const workerPollInterval = 5 * time.Second

type Executor interface {
	Execute(ctx context.Context) error
}

type WorkerPool struct {
	executor   Executor
	numWorkers int
}

func NewWorkerPool(executor Executor, numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = 3
	}
	return &WorkerPool{executor: executor, numWorkers: numWorkers}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < wp.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			wp.runWorker(ctx, workerID)
		}(i)
	}
	wg.Wait()
}

func (wp *WorkerPool) runWorker(ctx context.Context, id int) {
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := wp.executor.Execute(ctx); err != nil {
				fmt.Printf("worker %d: error executing job: %v\n", id, err)
			}
		}
	}
}
