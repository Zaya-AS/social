package likes_test

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"

	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/postgres"
	"gitlab.com/zayi-group/go-spirnt1/internal/queue/inmem"
)

func TestService_ApplyLikeDuplicate(t *testing.T) {
	fixture := newLikeFixture(t, 1)
	service := fixture.service
	user := fixture.user
	post := fixture.post

	for _, commandID := range []string{"cmd-1", "cmd-2"} {
		result := service.ApplyLike(context.Background(), domain.LikeCommand{
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

	assertStoredLikes(t, fixture, 2)
}

func TestService_EnqueueLikeCanceled(t *testing.T) {
	queue, err := inmem.NewQueue(1)
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}
	defer queue.Close()

	service := likes.NewService(
		nil,
		nil,
		queue,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)

	go func() {
		_, err := service.EnqueueLike(ctx, domain.LikeCommand{
			CommandID: "cmd-1",
			UserID:    1,
			PostID:    1,
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

type likeFixture struct {
	pool     *pgxpool.Pool
	postRepo *postgres.PostRepository
	user     *domain.User
	post     *domain.Post
	queue    *inmem.Queue
	service  *likes.Service
}

func newLikeFixture(tb testing.TB, capacity int) *likeFixture {
	tb.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		tb.Skip("TEST_DATABASE_URL is required; use a migrated PostgreSQL test database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		tb.Fatal("connect test PostgreSQL: check TEST_DATABASE_URL and database availability")
	}
	tb.Cleanup(pool.Close)

	userRepo := postgres.NewUserRepository(pool)
	postRepo := postgres.NewPostRepository(pool)
	author := &domain.User{Username: "test-author-" + uuid.NewString()}
	user := &domain.User{Username: "test-liker-" + uuid.NewString()}
	post := &domain.Post{Text: "test post"}

	tb.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		if _, err := pool.Exec(cleanupCtx, "DELETE FROM posts WHERE id = $1", post.ID); err != nil {
			tb.Errorf("delete fixture post: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1 OR id = $2", author.ID, user.ID); err != nil {
			tb.Errorf("delete fixture users: %v", err)
		}
	})

	if err := userRepo.Create(ctx, author); err != nil {
		tb.Fatalf("create fixture author: %v", err)
	}
	if err := userRepo.Create(ctx, user); err != nil {
		tb.Fatalf("create fixture liker: %v", err)
	}

	post.AuthorID = author.ID
	if err := postRepo.Create(ctx, post); err != nil {
		tb.Fatalf("create fixture post: %v", err)
	}

	queue, err := inmem.NewQueue(capacity)
	if err != nil {
		tb.Fatalf("create fixture queue: %v", err)
	}
	tb.Cleanup(queue.Close)

	return &likeFixture{
		pool:     pool,
		postRepo: postRepo,
		user:     user,
		post:     post,
		queue:    queue,
		service:  likes.NewService(userRepo, postRepo, queue),
	}
}

func assertStoredLikes(tb testing.TB, fixture *likeFixture, expected int) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	post, err := fixture.postRepo.GetByID(ctx, fixture.post.ID)
	if err != nil {
		tb.Fatalf("get fixture post: %v", err)
	}
	if post.LikesCount != expected {
		tb.Fatalf("expected %d likes including author like, got %d", expected, post.LikesCount)
	}

	var count int
	err = fixture.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM likes WHERE post_id = $1 AND user_id = $2",
		fixture.post.ID, fixture.user.ID,
	).Scan(&count)
	if err != nil {
		tb.Fatalf("count fixture user likes: %v", err)
	}
	if count != 1 {
		tb.Fatalf("expected one like from fixture user, got %d", count)
	}
}
