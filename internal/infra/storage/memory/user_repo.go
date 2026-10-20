package memory

import (
	"sync"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type UserRepo struct {
	mu        sync.RWMutex
	usersByID map[string]*domain.User
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		usersByID: make(map[string]*domain.User),
	}
}

func (r *UserRepo) Create(user *domain.User) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existingUser := range r.usersByID {
		if existingUser.Username == user.Username {
			return domain.ErrUsernameTaken
		}
	}

	r.usersByID[user.ID] = user

	return nil
}

func (r *UserRepo) GetByID(id string) (*domain.User, error) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.usersByID[id]

	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (r *UserRepo) List() ([]*domain.User, error) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*domain.User, 0)

	for _, user := range r.usersByID {
		users = append(users, user)
	}

	return users, nil
}
