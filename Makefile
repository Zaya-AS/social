.PHONY: up down test test-race itest migrate-up migrate-down db-reset

up:
	docker-compose up -d --build

down:
	docker-compose down

test:
	go test ./...

test-race:
	go test -race ./...

migrate-up:
	docker-compose run --rm migrate

migrate-down:
	docker-compose run --rm migrate -dir /migrations down

db-reset:
	docker-compose down
	docker volume rm social_postgres_data
	docker-compose up -d --build
itest:
	go test -tags=integration -count=1 -timeout=10m -v ./tests/integration
