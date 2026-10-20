package app

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"

	"github.com/google/uuid"
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

func (s *UserService) RegisterUser(username string) (*domain.User, error) {

	username = strings.TrimSpace(username)

	if username == "" {
		return nil, fmt.Errorf("%w : name", ErrInvalidRegistration)
	}

	user := &domain.User{
		ID:       uuid.NewString(),
		Username: username,
	}

	err := s.repo.Create(user)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) List() ([]*domain.User, error) {

	users, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return users, nil
}
