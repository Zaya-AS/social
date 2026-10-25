package likes

import (
	"context"
	"errors"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

var (
	ErrQueueClosed = errors.New("queue is closed")
	ErrQueueFull   = errors.New("queue is full")
)

type Job struct {
	Command domain.LikeCommand
	Result  chan domain.LikeResult
}

type Queue interface {
	Enqueue(Job) error
	Dequeue(context.Context) (Job, error)
	Close()
}
