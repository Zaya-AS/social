package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context) ([]*User, error)
}

type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	GetByID(ctx context.Context, id int64) (*Post, error)
	List(ctx context.Context) ([]*Post, error)
	LikePost(ctx context.Context, userID, postID int64) error
}
