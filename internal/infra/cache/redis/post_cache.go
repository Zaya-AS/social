package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type PostCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewPostCache(client *redis.Client, ttl time.Duration) *PostCache {
	return &PostCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *PostCache) Get(ctx context.Context, key string) ([]*domain.Post, bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get posts cache: %w", err)
	}

	posts := make([]*domain.Post, 0)
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, false, fmt.Errorf("decode posts cache: %w", err)
	}

	return posts, true, nil
}

func (c *PostCache) Set(ctx context.Context, key string, posts []*domain.Post) error {
	if posts == nil {
		posts = make([]*domain.Post, 0)
	}

	data, err := json.Marshal(posts)
	if err != nil {
		return fmt.Errorf("encode posts cache: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set posts cache: %w", err)
	}

	return nil
}

func (c *PostCache) Invalidate(ctx context.Context) error {
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "posts:list:v1:*", 100).Result()
		if err != nil {
			return fmt.Errorf("scan posts cache keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("invalidate posts cache: %w", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}
