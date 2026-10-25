package likes

import (
	"context"
	"errors"
	"sync"
)

type WorkerPool struct {
	queue     Queue
	service   *Service
	workers   int
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
	done      chan struct{}
}

func NewWorkerPool(queue Queue, service *Service, worker int) (*WorkerPool, error) {
	if worker < 1 {
		return nil, errors.New("worker count must be greater than zero")
	}
	return &WorkerPool{
		queue:   queue,
		service: service,
		workers: worker,
		done:    make(chan struct{}),
	}, nil
}

func (wp *WorkerPool) runWorker() {
	defer wp.wg.Done()

	for {
		job, err := wp.queue.Dequeue(context.Background())
		if err != nil {
			return
		}
		result := wp.service.ApplyLike(job.Command)
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
