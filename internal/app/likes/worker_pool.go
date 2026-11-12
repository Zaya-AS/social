package likes

import (
	"context"
	"errors"
	"sync"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/logging"
)

type LikeExecutor interface {
	ApplyLike(context.Context, domain.LikeCommand) *domain.LikeResult
}

type WorkerPool struct {
	queue            Queue
	executor         LikeExecutor
	workers          int
	wg               sync.WaitGroup
	startOnce        sync.Once
	stopOnce         sync.Once
	done             chan struct{}
	executionTimeuot time.Duration
}

func NewWorkerPool(queue Queue, executor LikeExecutor, worker int, executionTimeout time.Duration) (*WorkerPool, error) {
	if worker < 1 {
		return nil, errors.New("worker count must be greater than zero")
	}
	if executionTimeout <= 0 {
		return nil, errors.New("execution timeout must be greater than zero")
	}

	return &WorkerPool{
		queue:            queue,
		executor:         executor,
		workers:          worker,
		executionTimeuot: executionTimeout,
		done:             make(chan struct{}),
	}, nil
}

func (wp *WorkerPool) runWorker() {
	defer wp.wg.Done()

	for {
		job, err := wp.queue.Dequeue(context.Background())
		if err != nil {
			return
		}

		jobCtx := logging.WithRequestID(
			context.Background(),
			job.Command.RequestID,
		)

		ctx, cancel := context.WithTimeout(
			jobCtx,
			wp.executionTimeuot,
		)

		result := wp.executor.ApplyLike(ctx, job.Command)
		cancel()

		job.Result <- *result
	}
}

func (wp *WorkerPool) Start() {
	wp.startOnce.Do(func() {
		wp.wg.Add(wp.workers)
		for i := 0; i < wp.workers; i++ {
			go wp.runWorker()
		}
	})
}

func (wp *WorkerPool) Stop(ctx context.Context) error {

	wp.stopOnce.Do(func() {
		wp.Start()
		wp.queue.Close()

		go func() {
			wp.wg.Wait()
			close(wp.done)
		}()
	})

	select {

	case <-wp.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

}
