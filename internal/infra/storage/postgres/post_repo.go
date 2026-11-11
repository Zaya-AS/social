package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type PostRepository struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{
		pool: pool,
	}
}

func (r *PostRepository) Create(ctx context.Context, post *domain.Post) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create post transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	const query = `
		WITH new_post AS (
			INSERT INTO posts (author_id, text)
			VALUES ($1, $2)
			RETURNING id, author_id, created_at
		),
		new_like AS (
			INSERT INTO likes (post_id, user_id)
			SELECT id, author_id FROM new_post
			RETURNING post_id
		)
		SELECT p.id, p.created_at
		FROM new_post p
		JOIN new_like l ON l.post_id = p.id
	`

	var postID int64
	var createdAt time.Time

	err = tx.QueryRow(ctx, query, post.AuthorID, post.Text).Scan(
		&postID,
		&createdAt,
	)
	if err != nil {
		return fmt.Errorf("create post with author like: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create post transaction: %w", err)
	}

	post.ID = postID
	post.CreatedAt = createdAt
	post.Likes = []*domain.Like{
		{
			PostID: postID,
			UserID: post.AuthorID,
		},
	}
	post.LikesCount = 1

	return nil
}

func (r *PostRepository) GetByID(ctx context.Context, id int64) (*domain.Post, error) {

	const query = `
		SELECT 
			p.id,
			p.author_id,
			p.text,
			p.created_at, 
			COUNT(l.user_id) 
		FROM posts p
		LEFT JOIN likes l ON l.post_id = p.id
		WHERE p.id = $1
		GROUP BY p.id
	`
	var post domain.Post

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&post.ID,
		&post.AuthorID,
		&post.Text,
		&post.CreatedAt,
		&post.LikesCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPostNotFound
		}

		return nil, fmt.Errorf("get post by id: %w", err)
	}

	return &post, nil
}

func (r *PostRepository) List(ctx context.Context) ([]*domain.Post, error) {

	const query = `
	SELECT
		p.id,
		p.author_id,
		p.text,
		p.created_at,
		COUNT(l.user_id)
	FROM posts p
	LEFT JOIN likes l ON l.post_id = p.id
	GROUP BY p.id
	ORDER BY p.created_at DESC, p.id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.Post, 0)

	for rows.Next() {
		var post domain.Post

		if err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Text,
			&post.CreatedAt,
			&post.LikesCount,
		); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}

		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}

	return posts, nil
}

func (r *PostRepository) LikePost(ctx context.Context, userID, postID int64) error {

	const query = `
		INSERT INTO likes(post_id, user_id)
		VALUES($1,$2)
		ON CONFLICT (post_id, user_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, postID, userID)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			switch pgErr.ConstraintName {
			case "likes_post_id_fkey":
				return domain.ErrPostNotFound
			case "likes_user_id_fkey":
				return domain.ErrUserNotFound
			}
		}
		return fmt.Errorf("like post: %w", err)
	}
	return nil
}
