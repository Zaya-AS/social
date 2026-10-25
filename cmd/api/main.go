package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/app/likes"
	"gitlab.com/zayi-group/go-spirnt1/internal/config"
	httpapi "gitlab.com/zayi-group/go-spirnt1/internal/infra/http"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/memory"
	"gitlab.com/zayi-group/go-spirnt1/internal/queue/inmem"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env was not loaded: %v", err)
	}

	cfg, err := config.Load()

	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	userRepo := memory.NewUserRepo()
	userService := app.NewUserService(userRepo)
	userHandler := httpapi.NewUserHandler(userService)

	postRepo := memory.NewPostRepo()
	postService := app.NewPostService(postRepo, userRepo)
	likeQueue, err := inmem.NewQueue(100)
	if err != nil {
		return fmt.Errorf("create like queue: %w", err)
	}

	likeService := likes.NewService(userRepo, postRepo, likeQueue)

	workerPool, err := likes.NewWorkerPool(likeQueue, likeService, 4)
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

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
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
		10*time.Second,
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
		10*time.Second,
	)
	defer cancelWorkers()

	if err := workerPool.Stop(workerCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("stop worker pool: %w", err))
	}

	return runErr
}
