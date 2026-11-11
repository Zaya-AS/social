package domain

import "time"

type Post struct {
	ID         int64     `json:"id"`
	AuthorID   int64     `json:"authorID"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"createdAt"`
	Likes      []*Like   `json:"likes"`
	LikesCount int       `json:"likesCount"`
}
