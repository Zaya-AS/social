package domain

type UserRepository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	List() ([]*User, error)
}

type PostRepository interface {
	Create(post *Post) error
	GetByID(id string) (*Post, error)
	List() ([]*Post, error)
	LikePost(userID, postID string) error
}
