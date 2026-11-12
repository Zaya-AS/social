//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

func TestRegisterUser_UniqueUsername(t *testing.T) {
	f := newFixture(t)
	user, err := f.users.RegisterUser(f.ctx, "alice")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if user.ID <= 0 || user.Username != "alice" || user.CreatedAt.IsZero() {
		t.Fatalf("unexpected registered user: %+v", user)
	}
	if _, err := f.users.RegisterUser(f.ctx, "alice"); !errors.Is(err, domain.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}

	var count int
	if err := testEnv.pool.QueryRow(f.ctx, "SELECT COUNT(*) FROM users WHERE username = $1", "alice").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one stored user, got %d", count)
	}
}

func TestCreatePost_AuthorLikeAndList(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "first post")
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	stored, err := f.postRepo.GetByID(f.ctx, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AuthorID != author.ID || stored.Text != "first post" ||
		stored.CreatedAt.IsZero() || stored.LikesCount != 1 {
		t.Fatalf("unexpected stored post: %+v", stored)
	}
	assertCount(t, f, "SELECT COUNT(*) FROM likes WHERE post_id = $1 AND user_id = $2", 1, post.ID, author.ID)

	posts, err := f.postRepo.List(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].ID != post.ID || posts[0].LikesCount != 1 {
		t.Fatalf("unexpected posts list: %+v", posts)
	}
}

func TestLikePost_DuplicateDoesNotIncreaseCount(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	liker, err := f.users.RegisterUser(f.ctx, "liker")
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "post")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := f.postRepo.LikePost(f.ctx, liker.ID, post.ID); err != nil {
			t.Fatalf("like post, attempt %d: %v", i+1, err)
		}
	}
	assertCount(t, f, "SELECT COUNT(*) FROM likes WHERE post_id = $1 AND user_id = $2", 1, post.ID, liker.ID)
	posts, err := f.postRepo.List(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].LikesCount != 2 {
		t.Fatalf("expected two likes including author, got %+v", posts)
	}
}

func TestCreatePost_RollsBackWhenAuthorLikeFails(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}

	const trigger = "CREATE FUNCTION reject_test_like() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced author like failure' USING ERRCODE = '23514'; END; $$; CREATE TRIGGER reject_test_like BEFORE INSERT ON likes FOR EACH ROW EXECUTE FUNCTION reject_test_like();"
	if _, err := testEnv.pool.Exec(f.ctx, trigger); err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testEnv.pool.Exec(f.ctx, "DROP TRIGGER IF EXISTS reject_test_like ON likes; DROP FUNCTION IF EXISTS reject_test_like();"); err != nil {
			t.Errorf("remove failure trigger: %v", err)
		}
	})

	_, err = f.posts.CreatePost(f.ctx, author.ID, "must roll back")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("expected forced author like error, got %v", err)
	}
	assertCount(t, f, "SELECT COUNT(*) FROM posts", 0)
	assertCount(t, f, "SELECT COUNT(*) FROM likes", 0)
}

func TestLikePost_MissingReferences(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "post")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.postRepo.LikePost(f.ctx, author.ID, post.ID+1000); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
	if err := f.postRepo.LikePost(f.ctx, author.ID+1000, post.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func assertCount(t *testing.T, f *fixture, query string, want int, args ...any) {
	t.Helper()
	var count int
	if err := testEnv.pool.QueryRow(f.ctx, query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected count %d, got %d", want, count)
	}
}
