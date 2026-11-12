package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/logging"
)

var ErrInvalidPost = errors.New("invalid post data")

type PostService struct {
	postRepo domain.PostRepository
	userRepo domain.UserRepository
	cache    PostCache
}

func NewPostService(postRepo domain.PostRepository, userRepo domain.UserRepository, cache PostCache) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
		cache:    cache,
	}
}

func (s *PostService) CreatePost(ctx context.Context, authorID int64, text string) (*domain.Post, error) {
	text = strings.TrimSpace(text)

	if authorID <= 0 || text == "" {
		return nil, ErrInvalidPost
	}

	if _, err := s.userRepo.GetByID(ctx, authorID); err != nil {
		return nil, fmt.Errorf("find post author: %w", err)
	}

	post := &domain.Post{
		AuthorID: authorID,
		Text:     text,
	}

	err := s.postRepo.Create(ctx, post)

	if err != nil {
		return nil, err
	}
	if err := s.cache.Invalidate(ctx); err != nil {
		slog.WarnContext(ctx, "invalidate posts cache", "request_id", logging.RequestID(ctx), "error", err)
	}

	return post, nil
}

func (s *PostService) GetPost(ctx context.Context, postID int64) (*domain.Post, error) {

	post, err := s.postRepo.GetByID(ctx, postID)

	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) List(ctx context.Context) ([]*domain.Post, error) {
	const key = "posts:list:v1:all"

	cachedPosts, found, err := s.cache.Get(ctx, key)
	if err != nil {
		slog.WarnContext(ctx, "read posts cache", "request_id", logging.RequestID(ctx), "error", err)
	} else if found {
		return cachedPosts, nil
	}

	posts, err := s.postRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, posts); err != nil {
		slog.WarnContext(ctx, "write posts cache", "request_id", logging.RequestID(ctx), "error", err)
	}

	return posts, nil
}
