package likes

import (
	"context"
	"errors"

	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

var (
	ErrInvalidLike = errors.New("invalid data user or post")
)

type Service struct {
	userRepo domain.UserRepository
	postRepo domain.PostRepository
	queue    Queue
}

func NewService(userRepo domain.UserRepository, postRepo domain.PostRepository, queue Queue) *Service {
	return &Service{
		userRepo: userRepo,
		postRepo: postRepo,
		queue:    queue,
	}
}

func (s *Service) ApplyLike(ctx context.Context, cmd domain.LikeCommand) *domain.LikeResult {

	result := &domain.LikeResult{
		CommandID: cmd.CommandID,
	}

	if cmd.PostID <= 0 || cmd.UserID <= 0 {
		result.Err = ErrInvalidLike
		return result
	}

	if _, err := s.userRepo.GetByID(ctx, cmd.UserID); err != nil {
		result.Err = err
		return result
	}

	result.Err = s.postRepo.LikePost(ctx, cmd.UserID, cmd.PostID)
	return result
}

func (s *Service) EnqueueLike(ctx context.Context, cmd domain.LikeCommand) (domain.LikeResult, error) {

	if err := ctx.Err(); err != nil {
		return domain.LikeResult{}, err
	}

	resChan := make(chan domain.LikeResult, 1)

	job := Job{
		Command: cmd,
		Result:  resChan,
	}

	if err := s.queue.Enqueue(job); err != nil {
		return domain.LikeResult{}, err
	}

	select {
	case result := <-resChan:
		return result, nil
	case <-ctx.Done():
		return domain.LikeResult{}, ctx.Err()
	}
}
