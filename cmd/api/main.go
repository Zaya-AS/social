package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/config"
	rediscache "gitlab.com/zayi-group/go-spirnt1/internal/infra/cache/redis"
	httpapi "gitlab.com/zayi-group/go-spirnt1/internal/infra/http"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/postgres"
	"gitlab.com/zayi-group/go-spirnt1/internal/queue/inmem"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {

	if err := godotenv.Load(); err != nil {
		slog.Warn("environment file was not loaded", "error", err)
	}

	cfg, err := config.Load()

	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	connectCtx, cancelConn := context.WithTimeout(
		context.Background(),
		cfg.ConnectTimeout,
	)

	pool, err := postgres.NewPool(connectCtx, cfg.DatabaseURL)
	cancelConn()

	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()

	redisCtx, cancelRedis := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	redisClient, err := rediscache.NewClient(redisCtx, cfg.RedisAddr, cfg.RedisDB)
	cancelRedis()

	if err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("close Redis", "error", err)
		}
	}()

	userRepo := postgres.NewUserRepository(pool)
	userService := app.NewUserService(userRepo)
	userHandler := httpapi.NewUserHandler(userService)

	postRepo := postgres.NewPostRepository(pool)
	postCache := rediscache.NewPostCache(redisClient, cfg.RedisCacheTTL)
	postService := app.NewPostService(postRepo, userRepo, postCache)
	likeQueue, err := inmem.NewQueue(cfg.QueueCapacity)
	if err != nil {
		return fmt.Errorf("create like queue: %w", err)
	}

	likeService := likes.NewService(userRepo, postRepo, likeQueue)

	workerPool, err := likes.NewWorkerPool(likeQueue, likeService, cfg.WorkerCount, cfg.LikeExecutionTimeout)
	if err != nil {
		return fmt.Errorf("create worker pool: %w", err)
	}
	workerPool.Start()

	postHandler := httpapi.NewPostHandler(postService, likeService)

	httpRouter := httpapi.NewRouter(userHandler, postHandler)

	address := ":" + cfg.HTTPPort

	server := &http.Server{
		Addr:    address,
		Handler: httpRouter,
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	var runErr error

	select {
	case <-signalCtx.Done():
		stopSignals()
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("run HTTP server: %w", err)
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		cfg.HTTPShutdownTimeout,
	)
	shutdownErr := server.Shutdown(shutdownCtx)
	cancelShutdown()

	if shutdownErr != nil {
		runErr = errors.Join(
			runErr,
			fmt.Errorf("shutdown HTTP server: %w", shutdownErr),
		)

		if err := server.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close HTTP server: %w", err))
		}
	}

	workerCtx, cancelWorkers := context.WithTimeout(
		context.Background(),
		cfg.WorkerShutdownTimeout,
	)
	defer cancelWorkers()

	if err := workerPool.Stop(workerCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("stop worker pool: %w", err))
	}

	return runErr
}
