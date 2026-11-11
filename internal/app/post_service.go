package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

var ErrInvalidPost = errors.New("invalid post data")

type PostService struct {
	postRepo domain.PostRepository
	userRepo domain.UserRepository
}

func NewPostService(postRepo domain.PostRepository, userRepo domain.UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
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

	posts, err := s.postRepo.List(ctx)

	if err != nil {
		return nil, err
	}

	return posts, nil
}
