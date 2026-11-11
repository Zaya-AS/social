package domain

import "time"

type Like struct {
	UserID int64
	PostID int64
}

type LikeCommand struct {
	CommandID     string
	PostID        int64
	UserID        int64
	CorrelationID string
	RequestID     string
	CreatedAt     time.Time
}

type LikeResult struct {
	CommandID string
	Err       error
}
