package memory

import (
	"sync"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type PostRepo struct {
	mu        sync.RWMutex
	postsByID map[string]*domain.Post
}

func NewPostRepo() *PostRepo {
	return &PostRepo{
		postsByID: make(map[string]*domain.Post),
	}
}

func (r *PostRepo) Create(post *domain.Post) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.postsByID[post.ID]

	if ok {
		return domain.ErrPostAlreadyExists
	}

	r.postsByID[post.ID] = post

	return nil
}

func (r *PostRepo) GetByID(id string) (*domain.Post, error) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.postsByID[id]
	if !ok {
		return nil, domain.ErrPostNotFound
	}

	postCopy := *post

	postCopy.Likes = make([]*domain.Like, len(post.Likes))
	copy(postCopy.Likes, post.Likes)

	return &postCopy, nil
}

func (r *PostRepo) List() ([]*domain.Post, error) {

	r.mu.RLock()
	defer r.mu.RUnlock()

	posts := make([]*domain.Post, 0)

	for _, post := range r.postsByID {
		postCopy := *post
		postCopy.Likes = make([]*domain.Like, len(postCopy.Likes))
		copy(postCopy.Likes, post.Likes)
		posts = append(posts, &postCopy)
	}

	return posts, nil
}

func (r *PostRepo) LikePost(userID, postID string) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.postsByID[postID]

	if !ok {
		return domain.ErrPostNotFound
	}

	for _, like := range post.Likes {
		if like.UserID == userID {
			return nil
		}
	}

	post.Likes = append(post.Likes, &domain.Like{
		UserID: userID,
		PostID: postID,
	})

	return nil
}
