package likes_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

func BenchmarkLike_Direct(b *testing.B) {
	fixture := newLikeFixture(b, 1)
	service := fixture.service

	cmd := domain.LikeCommand{
		CommandID: "benchmark-command",
		UserID:    fixture.user.ID,
		PostID:    fixture.post.ID,
	}

	if result := service.ApplyLike(context.Background(), cmd); result.Err != nil {
		b.Fatal(result.Err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := service.ApplyLike(context.Background(), cmd)
		if result.Err != nil {
			b.Fatal(result.Err)
		}
	}
	b.StopTimer()
}

func BenchmarkLike_QueueWorkerPool(b *testing.B) {
	fixture := newLikeFixture(b, 100)
	service := fixture.service
	pool, err := likes.NewWorkerPool(fixture.queue, service, 4, 5*time.Second)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := pool.Stop(ctx); err != nil {
			b.Errorf("stop worker pool: %v", err)
		}
	})

	pool.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := domain.LikeCommand{
		CommandID: "benchmark-command",
		UserID:    fixture.user.ID,
		PostID:    fixture.post.ID,
	}

	result, err := service.EnqueueLike(ctx, cmd)
	if err != nil {
		b.Fatal(err)
	}
	if result.Err != nil {
		b.Fatal(result.Err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := service.EnqueueLike(ctx, cmd)
		if err != nil {
			b.Fatal(err)
		}
		if result.Err != nil {
			b.Fatal(result.Err)
		}
	}

	b.StopTimer()
}
