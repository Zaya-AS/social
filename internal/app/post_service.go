package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

var (
	ErrInvalidPost = errors.New("invalid post data")
	ErrInvalidLike = errors.New("invalid like data")
)

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

func (s *PostService) CreatePost(authorID, text string) (*domain.Post, error) {

	authorID = strings.TrimSpace(authorID)
	text = strings.TrimSpace(text)

	if authorID == "" || text == "" {
		return nil, ErrInvalidPost
	}

	if _, err := s.userRepo.GetByID(authorID); err != nil {
		return nil, fmt.Errorf("not find author: %w", err)
	}

	post := &domain.Post{
		ID:        uuid.NewString(),
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: time.Now(),
	}

	err := s.postRepo.Create(post)

	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) GetPost(postID string) (*domain.Post, error) {

	post, err := s.postRepo.GetByID(postID)

	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) List() ([]*domain.Post, error) {

	posts, err := s.postRepo.List()

	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *PostService) LikePost(userID, postID string) error {

	userID = strings.TrimSpace(userID)
	postID = strings.TrimSpace(postID)

	if userID == "" || postID == "" {
		return ErrInvalidLike
	}

	if _, err := s.userRepo.GetByID(userID); err != nil {
		return fmt.Errorf("find user for like: %w", err)
	}

	err := s.postRepo.LikePost(userID, postID)

	if err != nil {
		return err
	}

	return nil
}
