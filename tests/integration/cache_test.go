//go:build integration

package integration_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	rediscache "gitlab.com/zayi-group/go-spirnt1/internal/infra/cache/redis"
)

const postsKey = "posts:list:v1:all"

func TestPostCache_MissRoundTripAndExpiration(t *testing.T) {
	f := newFixture(t)
	if _, found, err := f.cache.Get(f.ctx, postsKey); err != nil || found {
		t.Fatalf("expected cache miss, found=%v err=%v", found, err)
	}
	posts := []*domain.Post{{ID: 42, AuthorID: 7, Text: "cached post", LikesCount: 3}}
	if err := f.cache.Set(f.ctx, postsKey, posts); err != nil {
		t.Fatal(err)
	}
	got, found, err := f.cache.Get(f.ctx, postsKey)
	if err != nil || !found {
		t.Fatalf("expected cache hit, found=%v err=%v", found, err)
	}
	if !reflect.DeepEqual(got, posts) {
		t.Fatalf("unexpected cached posts: %+v", got)
	}

	ttl, err := testEnv.redis.PTTL(f.ctx, postsKey).Result()
	if err != nil {
		t.Fatal(err)
	}
	if ttl <= 0 || ttl > 5*time.Second {
		t.Fatalf("expected positive TTL at most 5s, got %s", ttl)
	}

	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		_, found, err := f.cache.Get(f.ctx, postsKey)
		if err != nil {
			t.Fatal(err)
		}
		if !found {
			break
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("cache entry did not expire")
		case <-f.ctx.Done():
			t.Fatal(f.ctx.Err())
		}
	}
}

func TestPostCache_EmptyListAndInvalidation(t *testing.T) {
	f := newFixture(t)
	if err := f.cache.Set(f.ctx, postsKey, nil); err != nil {
		t.Fatal(err)
	}
	posts, found, err := f.cache.Get(f.ctx, postsKey)
	if err != nil || !found || posts == nil || len(posts) != 0 {
		t.Fatalf("expected cached empty slice, posts=%v found=%v err=%v", posts, found, err)
	}

	for i := 0; i < 205; i++ {
		if err := f.cache.Set(f.ctx, fmt.Sprintf("posts:list:v1:offset=%d", i), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := testEnv.redis.Set(f.ctx, "unrelated:key", "keep", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := f.cache.Invalidate(f.ctx); err != nil {
		t.Fatal(err)
	}
	keys, err := testEnv.redis.Keys(f.ctx, "posts:list:v1:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected all list keys removed, remaining=%d", len(keys))
	}
	if value, err := testEnv.redis.Get(f.ctx, "unrelated:key").Result(); err != nil || value != "keep" {
		t.Fatalf("unrelated key was changed: value=%q err=%v", value, err)
	}
}

func TestPostService_ListUsesCache(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "original")
	if err != nil {
		t.Fatal(err)
	}
	posts, err := f.posts.List(f.ctx)
	if err != nil || len(posts) != 1 {
		t.Fatalf("list posts: posts=%v err=%v", posts, err)
	}
	if _, found, err := f.cache.Get(f.ctx, postsKey); err != nil || !found {
		t.Fatalf("list did not populate cache: found=%v err=%v", found, err)
	}

	if _, err := testEnv.pool.Exec(f.ctx, "UPDATE posts SET text = $1 WHERE id = $2", "changed in database", post.ID); err != nil {
		t.Fatal(err)
	}
	posts, err = f.posts.List(f.ctx)
	if err != nil || len(posts) != 1 || posts[0].Text != "original" {
		t.Fatalf("expected original value from cache, posts=%v err=%v", posts, err)
	}

	if err := f.cache.Invalidate(f.ctx); err != nil {
		t.Fatal(err)
	}
	posts, err = f.posts.List(f.ctx)
	if err != nil || len(posts) != 1 || posts[0].Text != "changed in database" {
		t.Fatalf("expected fresh database value, posts=%v err=%v", posts, err)
	}
}

func TestPostService_CreateInvalidatesCache(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.posts.List(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, found, err := f.cache.Get(f.ctx, postsKey); err != nil || !found {
		t.Fatalf("expected initial cache entry: found=%v err=%v", found, err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "new post")
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := f.cache.Get(f.ctx, postsKey); err != nil || found {
		t.Fatalf("expected invalidation after creation: found=%v err=%v", found, err)
	}
	posts, err := f.posts.List(f.ctx)
	if err != nil || len(posts) != 1 || posts[0].ID != post.ID {
		t.Fatalf("expected new post in fresh list: posts=%v err=%v", posts, err)
	}
}

func TestPostService_CorruptCacheFallsBackToDatabase(t *testing.T) {
	f := newFixture(t)
	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.CreatePost(f.ctx, author.ID, "post")
	if err != nil {
		t.Fatal(err)
	}
	if err := testEnv.redis.Set(f.ctx, postsKey, "invalid JSON", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	posts, err := f.posts.List(f.ctx)
	if err != nil || len(posts) != 1 || posts[0].ID != post.ID {
		t.Fatalf("expected database fallback: posts=%v err=%v", posts, err)
	}
	cached, found, err := f.cache.Get(f.ctx, postsKey)
	if err != nil || !found || len(cached) != 1 || cached[0].ID != post.ID {
		t.Fatalf("expected repaired cache: posts=%v found=%v err=%v", cached, found, err)
	}
}

func TestPostService_UnavailableCacheDoesNotFailDatabaseOperations(t *testing.T) {
	f := newFixture(t)
	client := redis.NewClient(&redis.Options{Addr: testEnv.redis.Options().Addr})
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	cache := rediscache.NewPostCache(client, 5*time.Second)
	service := app.NewPostService(f.postRepo, f.userRepo, cache)

	author, err := f.users.RegisterUser(f.ctx, "author")
	if err != nil {
		t.Fatal(err)
	}
	post, err := service.CreatePost(f.ctx, author.ID, "post without cache")
	if err != nil {
		t.Fatalf("cache invalidation error must not fail creation: %v", err)
	}
	posts, err := service.List(f.ctx)
	if err != nil || len(posts) != 1 || posts[0].ID != post.ID {
		t.Fatalf("cache errors must not fail listing: posts=%v err=%v", posts, err)
	}
}
