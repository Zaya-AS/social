package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/logging"
)

func RequestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()

		requestCtx := logging.WithRequestID(ctx.Request.Context(), requestID)
		ctx.Request = ctx.Request.WithContext(requestCtx)

		ctx.Set("request_id", requestID)
		ctx.Header("X-Request-ID", requestID)

		ctx.Next()

		level := slog.LevelInfo
		status := ctx.Writer.Status()

		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if status >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		slog.Log(
			ctx.Request.Context(),
			level,
			"http request completed",
			"request_id", requestID,
			"method", ctx.Request.Method,
			"endpoint", ctx.FullPath(),
			"path", ctx.Request.URL.Path,
			"status", status,
			"latency_ms", float64(time.Since(start).Microseconds())/1000,
		)

	}
}
