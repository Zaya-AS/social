package domain

import "time"

type Like struct {
	UserID string
	PostID string
}

type LikeCommand struct {
	CommandID     string
	PostID        string
	UserID        string
	CorrelationID string
	RequestID     string
	CreatedAt     time.Time
}

type LikeResult struct {
	CommandID string
	Err       error
}
