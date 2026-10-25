package inmem

import (
	"context"
	"errors"
	"sync"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
)

type Queue struct {
	jobs   chan likes.Job
	mu     sync.Mutex
	closed bool
}

func NewQueue(capacity int) (*Queue, error) {
	if capacity <= 0 {
		return nil, errors.New("queue capacity must be greater than zero")
	}
	return &Queue{
		jobs: make(chan likes.Job, capacity),
	}, nil
}

func (q *Queue) Enqueue(job likes.Job) error {

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return likes.ErrQueueClosed
	}

	select {
	case q.jobs <- job:
		return nil
	default:
		return likes.ErrQueueFull
	}
}

func (q *Queue) Close() {

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}
	q.closed = true
	close(q.jobs)
}

func (q *Queue) Dequeue(ctx context.Context) (likes.Job, error) {

	select {
	case job, ok := <-q.jobs:
		if !ok {
			return likes.Job{}, likes.ErrQueueClosed
		}
		return job, nil
	case <-ctx.Done():
		return likes.Job{}, ctx.Err()
	}

}
