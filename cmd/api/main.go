package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	"gitlab.com/zayi-group/go-spirnt1/internal/config"
	httpapi "gitlab.com/zayi-group/go-spirnt1/internal/infra/http"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/memory"
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
	postHandler := httpapi.NewPostHandler(postService)

	httpRouter := httpapi.NewRouter(userHandler, postHandler)

	address := ":" + cfg.HTTPPort

	if err := httpRouter.Run(address); err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	return nil
}
