package likes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/memory"
	"gitlab.com/zayi-group/go-spirnt1/internal/queue/inmem"
)

func TestService_ApplyLikeDuplicate(t *testing.T) {
	userRepo := memory.NewUserRepo()
	postRepo := memory.NewPostRepo()

	user := &domain.User{
		ID:       "user-1",
		Username: "ivan",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	post := &domain.Post{
		ID:       "post-1",
		AuthorID: user.ID,
		Text:     "test post",
	}
	if err := postRepo.Create(post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	queue, err := inmem.NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	service := likes.NewService(userRepo, postRepo, queue)

	for _, commandID := range []string{"cmd-1", "cmd-2"} {
		result := service.ApplyLike(domain.LikeCommand{
			CommandID: commandID,
			UserID:    user.ID,
			PostID:    post.ID,
		})

		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Err != nil {
			t.Fatalf("apply like: %v", result.Err)
		}
		if result.CommandID != commandID {
			t.Fatalf("expected command ID %q, got %q",
				commandID, result.CommandID)
		}
	}

	saved, err := postRepo.GetByID(post.ID)
	if err != nil {
		t.Fatalf("get post: %v", err)
	}

	if len(saved.Likes) != 1 {
		t.Fatalf("expected 1 like, got %d", len(saved.Likes))
	}
	if saved.Likes[0].UserID != user.ID ||
		saved.Likes[0].PostID != post.ID {
		t.Fatal("saved like has incorrect user or post ID")
	}
}

func TestService_EnqueueLikeCanceled(t *testing.T) {
	queue, err := inmem.NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	service := likes.NewService(
		memory.NewUserRepo(),
		memory.NewPostRepo(),
		queue,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)

	go func() {
		_, err := service.EnqueueLike(ctx, domain.LikeCommand{
			CommandID: "cmd-1",
			UserID:    "user-1",
			PostID:    "post-1",
		})
		result <- err
	}()

	readCtx, cancelRead := context.WithTimeout(
		context.Background(), 5*time.Second,
	)
	defer cancelRead()

	job, err := queue.Dequeue(readCtx)
	if err != nil {
		t.Fatalf("dequeue job: %v", err)
	}

	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-readCtx.Done():
		t.Fatal("EnqueueLike did not finish after cancellation")
	}

	select {
	case job.Result <- domain.LikeResult{
		CommandID: job.Command.CommandID,
	}:
	default:
		t.Fatal("result channel blocks after cancellation")
	}
}
