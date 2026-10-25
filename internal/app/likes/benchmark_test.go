package likes_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/memory"
	"gitlab.com/zayi-group/go-spirnt1/internal/queue/inmem"
)

func BenchmarkLike_Direct(b *testing.B) {
	userRepo := memory.NewUserRepo()
	postRepo := memory.NewPostRepo()

	if err := userRepo.Create(&domain.User{
		ID:       "user-1",
		Username: "ivan",
	}); err != nil {
		b.Fatal(err)
	}

	if err := postRepo.Create(&domain.Post{
		ID:       "post-1",
		AuthorID: "user-1",
		Text:     "benchmark post",
	}); err != nil {
		b.Fatal(err)
	}

	queue, err := inmem.NewQueue(1)
	if err != nil {
		b.Fatal(err)
	}
	defer queue.Close()

	service := likes.NewService(userRepo, postRepo, queue)

	cmd := domain.LikeCommand{
		CommandID: "benchmark-command",
		UserID:    "user-1",
		PostID:    "post-1",
	}

	if result := service.ApplyLike(cmd); result.Err != nil {
		b.Fatal(result.Err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := service.ApplyLike(cmd)
		if result.Err != nil {
			b.Fatal(result.Err)
		}
	}
}

func BenchmarkLike_QueueWorkerPool(b *testing.B) {
	userRepo := memory.NewUserRepo()
	postRepo := memory.NewPostRepo()

	if err := userRepo.Create(&domain.User{
		ID:       "user-1",
		Username: "ivan",
	}); err != nil {
		b.Fatal(err)
	}

	if err := postRepo.Create(&domain.Post{
		ID:       "post-1",
		AuthorID: "user-1",
		Text:     "benchmark post",
	}); err != nil {
		b.Fatal(err)
	}

	queue, err := inmem.NewQueue(100)
	if err != nil {
		b.Fatal(err)
	}

	service := likes.NewService(userRepo, postRepo, queue)
	pool, err := likes.NewWorkerPool(queue, service, 4)
	if err != nil {
		queue.Close()
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
		UserID:    "user-1",
		PostID:    "post-1",
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
