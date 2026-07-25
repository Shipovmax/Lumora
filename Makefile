-include .env
export

GOOSE_DSN := "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)"

.PHONY: run-api run-worker build test vet up down logs migrate-up migrate-down

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

up:
	docker compose -f deployments/docker-compose.yml up --build -d

down:
	docker compose -f deployments/docker-compose.yml down

logs:
	docker compose -f deployments/docker-compose.yml logs -f

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres $(GOOSE_DSN) up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres $(GOOSE_DSN) down
