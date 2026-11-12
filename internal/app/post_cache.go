package app

import (
	"context"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type PostCache interface {
	Get(ctx context.Context, key string) ([]*domain.Post, bool, error)
	Set(ctx context.Context, key string, posts []*domain.Post) error
	Invalidate(ctx context.Context) error
}
