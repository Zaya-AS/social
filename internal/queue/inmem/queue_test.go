package inmem

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

func TestQueue_EnqueueFull(t *testing.T) {
	queue, err := NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	first := likes.Job{
		Command: domain.LikeCommand{CommandID: "first"},
	}
	if err := queue.Enqueue(first); err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}

	second := likes.Job{
		Command: domain.LikeCommand{CommandID: "second"},
	}
	err = queue.Enqueue(second)
	if !errors.Is(err, likes.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestQueue_Close(t *testing.T) {
	queue, err := NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	job := likes.Job{
		Command: domain.LikeCommand{CommandID: "first"},
	}
	if err := queue.Enqueue(job); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	queue.Close()
	queue.Close()

	if err := queue.Enqueue(job); !errors.Is(err, likes.ErrQueueClosed) {
		t.Fatalf("expected ErrQueueClosed, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue accepted job: %v", err)
	}
	if got.Command.CommandID != job.Command.CommandID {
		t.Fatalf("expected command %q, got %q",
			job.Command.CommandID, got.Command.CommandID)
	}

	_, err = queue.Dequeue(ctx)
	if !errors.Is(err, likes.ErrQueueClosed) {
		t.Fatalf("expected ErrQueueClosed after draining, got %v", err)
	}

}
func TestQueue_DequeueCanceled(t *testing.T) {
	queue, err := NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = queue.Dequeue(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
