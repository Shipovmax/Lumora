-include .env
export

.PHONY: run-api run-worker build test vet up down logs migrate-up migrate-down sqlc

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

vet:
	go vet ./...

up:
	docker compose -f deployments/docker-compose.yml up --build -d

down:
	docker compose -f deployments/docker-compose.yml down

logs:
	docker compose -f deployments/docker-compose.yml logs -f

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
