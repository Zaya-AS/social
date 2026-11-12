//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"gitlab.com/zayi-group/go-spirnt1/internal/app"
	rediscache "gitlab.com/zayi-group/go-spirnt1/internal/infra/cache/redis"
	"gitlab.com/zayi-group/go-spirnt1/internal/infra/storage/postgres"
)

var testEnv integrationEnvironment

type integrationEnvironment struct {
	pool       *pgxpool.Pool
	redis      *redis.Client
	containers []string
	network    string
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	defer testEnv.close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if err := testEnv.start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		return 1
	}
	return m.Run()
}

func docker(ctx context.Context, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker command failed: %w\n%s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func (env *integrationEnvironment) start(ctx context.Context) error {
	if _, err := docker(ctx, "info", "--format", "{{.ServerVersion}}"); err != nil {
		return fmt.Errorf("Docker must be installed and running: %w", err)
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return err
	}
	suffix := uuid.NewString()
	env.network = "social-it-" + suffix
	if _, err := docker(ctx, "network", "create", env.network); err != nil {
		env.network = ""
		return err
	}

	pgName := "social-it-pg-" + suffix
	env.containers = append(env.containers, pgName)
	if _, err := docker(ctx, "run", "--detach", "--name", pgName,
		"--network", env.network, "--publish", "127.0.0.1::5432",
		"--env", "POSTGRES_USER=integration",
		"--env", "POSTGRES_PASSWORD=integration",
		"--env", "POSTGRES_DB=integration",
		"postgres:17-alpine"); err != nil {
		return err
	}

	redisName := "social-it-redis-" + suffix
	env.containers = append(env.containers, redisName)
	if _, err := docker(ctx, "run", "--detach", "--name", redisName,
		"--network", env.network, "--publish", "127.0.0.1::6379",
		"redis:7.4-alpine"); err != nil {
		return err
	}

	pgAddr, err := publishedAddress(ctx, pgName, "5432/tcp")
	if err != nil {
		return err
	}
	redisAddr, err := publishedAddress(ctx, redisName, "6379/tcp")
	if err != nil {
		return err
	}

	env.pool, err = pgxpool.New(ctx, "postgres://integration:integration@"+pgAddr+"/integration?sslmode=disable")
	if err != nil {
		return err
	}
	env.redis = redis.NewClient(&redis.Options{Addr: redisAddr, MaxRetries: -1})

	if err := waitReady(ctx, env.pool.Ping); err != nil {
		return fmt.Errorf("wait for PostgreSQL: %w", err)
	}
	if err := waitReady(ctx, func(ctx context.Context) error {
		return env.redis.Ping(ctx).Err()
	}); err != nil {
		return fmt.Errorf("wait for Redis: %w", err)
	}

	const image = "social-integration-migrate:local"
	if _, err := docker(ctx, "build", "--tag", image,
		"--file", filepath.Join(root, "Dockerfile.migrate"), root); err != nil {
		return fmt.Errorf("build migration image: %w", err)
	}

	migrateName := "social-it-migrate-" + suffix
	env.containers = append(env.containers, migrateName)
	_, err = docker(ctx, "run", "--rm", "--name", migrateName,
		"--network", env.network,
		"--env", "GOOSE_DRIVER=postgres",
		"--env", "GOOSE_DBSTRING=postgres://integration:integration@"+pgName+":5432/integration?sslmode=disable",
		"--mount", "type=bind,source="+filepath.Join(root, "migrations")+",target=/migrations,readonly",
		image, "-dir", "/migrations", "up")
	if err != nil {
		return fmt.Errorf("apply goose migrations: %w", err)
	}
	env.containers = env.containers[:len(env.containers)-1]
	return nil
}

func publishedAddress(ctx context.Context, container, port string) (string, error) {
	output, err := docker(ctx, "port", container, port)
	if err != nil {
		return "", err
	}
	addr := strings.TrimSpace(strings.Split(output, "\n")[0])
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return "", fmt.Errorf("invalid published address %q: %w", addr, err)
	}
	return addr, nil
}

func waitReady(ctx context.Context, ping func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		attempt, cancelAttempt := context.WithTimeout(ctx, time.Second)
		lastErr = ping(attempt)
		cancelAttempt()
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func (env *integrationEnvironment) close() {
	if env.redis != nil {
		_ = env.redis.Close()
	}
	if env.pool != nil {
		env.pool.Close()
	}
	for i := len(env.containers) - 1; i >= 0; i-- {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if _, err := docker(ctx, "rm", "--force", "--volumes", env.containers[i]); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup test container:", err)
		}
		cancel()
	}
	if env.network != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := docker(ctx, "network", "rm", env.network); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup test network:", err)
		}
	}
}

type fixture struct {
	ctx      context.Context
	users    *app.UserService
	posts    *app.PostService
	userRepo *postgres.UserRepository
	postRepo *postgres.PostRepository
	cache    *rediscache.PostCache
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	if _, err := testEnv.pool.Exec(ctx, "TRUNCATE likes, posts, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("reset isolated test database: %v", err)
	}
	if err := testEnv.redis.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("reset isolated test Redis: %v", err)
	}

	userRepo := postgres.NewUserRepository(testEnv.pool)
	postRepo := postgres.NewPostRepository(testEnv.pool)
	cache := rediscache.NewPostCache(testEnv.redis, 5*time.Second)
	return &fixture{
		ctx:      ctx,
		users:    app.NewUserService(userRepo),
		posts:    app.NewPostService(postRepo, userRepo, cache),
		userRepo: userRepo,
		postRepo: postRepo,
		cache:    cache,
	}
}
