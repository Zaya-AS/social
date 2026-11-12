package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/logging"
)

func logDBError(ctx context.Context, operation string, err error) {
	fields := []any{
		"request_id", logging.RequestID(ctx),
		"operation", operation,
		"error", err,
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		fields = append(fields,
			"code", pgErr.Code,
			"constraint", pgErr.ConstraintName,
		)
	}
	slog.ErrorContext(ctx, "database operation failed", fields...)
}
