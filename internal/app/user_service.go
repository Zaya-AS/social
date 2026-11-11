package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

var (
	ErrInvalidRegistration = errors.New("invalid registration data")
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, username string) (*domain.User, error) {

	username = strings.TrimSpace(username)

	if username == "" {
		return nil, fmt.Errorf("%w : name", ErrInvalidRegistration)
	}

	user := &domain.User{
		Username: username,
	}

	err := s.repo.Create(ctx, user)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {

	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}
