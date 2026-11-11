package likes_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

func TestWorkerPool_ConcurrentDuplicateLikes(t *testing.T) {
	const requests = 20

	fixture := newLikeFixture(t, requests)
	queue := fixture.queue
	service := fixture.service
	pool, err := likes.NewWorkerPool(queue, service, 4, 5*time.Second)
	if err != nil {
		t.Fatalf("create worker pool: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pool.Stop(ctx); err != nil {
			t.Errorf("stop worker pool during cleanup: %v", err)
		}
	})

	pool.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errs := make(chan error, requests)
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		commandID := fmt.Sprintf("command-%d", i)

		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			result, err := service.EnqueueLike(ctx, domain.LikeCommand{
				CommandID: id,
				UserID:    fixture.user.ID,
				PostID:    fixture.post.ID,
				CreatedAt: time.Now(),
			})
			if err != nil {
				errs <- fmt.Errorf("%s: enqueue like: %w", id, err)
				return
			}
			if result.Err != nil {
				errs <- fmt.Errorf("%s: apply like: %w", id, result.Err)
				return
			}
			if result.CommandID != id {
				errs <- fmt.Errorf("expected command ID %q, got %q",
					id, result.CommandID)
			}
		}(commandID)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	assertStoredLikes(t, fixture, 2)

	stopCtx, stopCancel := context.WithTimeout(
		context.Background(), 5*time.Second,
	)
	defer stopCancel()

	if err := pool.Stop(stopCtx); err != nil {
		t.Fatalf("stop worker pool: %v", err)
	}
}

func TestWorkerPool_StopDrainsQueue(t *testing.T) {
	const commands = 5

	fixture := newLikeFixture(t, commands)
	queue := fixture.queue
	service := fixture.service
	pool, err := likes.NewWorkerPool(queue, service, 2, 5*time.Second)
	if err != nil {
		t.Fatalf("create worker pool: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pool.Stop(ctx); err != nil {
			t.Errorf("cleanup worker pool: %v", err)
		}
	})

	jobs := make([]likes.Job, 0, commands)

	for i := 0; i < commands; i++ {
		job := likes.Job{
			Command: domain.LikeCommand{
				CommandID: fmt.Sprintf("cmd-%d", i),
				UserID:    fixture.user.ID,
				PostID:    fixture.post.ID,
			},
			Result: make(chan domain.LikeResult, 1),
		}

		if err := queue.Enqueue(job); err != nil {
			t.Fatalf("enqueue job: %v", err)
		}
		jobs = append(jobs, job)
	}

	pool.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Stop(ctx); err != nil {
		t.Fatalf("stop worker pool: %v", err)
	}

	for _, job := range jobs {
		select {
		case result := <-job.Result:
			if result.Err != nil {
				t.Errorf("command %q: %v", result.CommandID, result.Err)
			}
			if result.CommandID != job.Command.CommandID {
				t.Errorf("expected command %q, got %q",
					job.Command.CommandID, result.CommandID)
			}
		default:
			t.Errorf("missing result for command %q", job.Command.CommandID)
		}
	}

	err = queue.Enqueue(likes.Job{})
	if !errors.Is(err, likes.ErrQueueClosed) {
		t.Errorf("expected ErrQueueClosed, got %v", err)
	}

	if err := pool.Stop(ctx); err != nil {
		t.Fatalf("repeated stop: %v", err)
	}
}
