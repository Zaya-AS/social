package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/zayi-group/go-spirnt1/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {

	const query = `
		INSERT INTO users (username)
		VALUES($1)
		RETURNING id, username,created_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		user.Username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_username_key" {
			return domain.ErrUsernameTaken
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {

	const query = `
		SELECT id, username, created_at FROM users
		WHERE id = $1
	`

	user := &domain.User{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {

	const query = `
		SELECT id, username, created_at FROM users
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)

	for rows.Next() {
		var user domain.User

		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan users: %w", err)
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}
