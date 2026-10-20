package domain

import "time"

type Post struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"authorID"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	Likes     []*Like   `json:"likes"`
}
