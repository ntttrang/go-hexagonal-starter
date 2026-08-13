APP_NAME := go-hexagonal-starter
MODULE := github.com/nttttranggo-hexagonal-starter
BIN := bin/api
GO ?= go

.PHONY: help tidy build run test test-integration lint gosec govulncheck trivy swag migrate-up migrate-down up down logs docker-build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

tidy: ## Download and tidy Go modules
	$(GO) mod tidy

build: ## Build the API binary
	$(GO) build -o $(BIN) ./cmd/api

run: ## Run the API locally
	$(GO) run ./cmd/api

test: ## Run unit tests
	$(GO) test ./... -count=1 -race -coverprofile=coverage.out -covermode=atomic

test-integration: ## Run integration tests (requires Postgres; set TEST_DATABASE_URL)
	$(GO) test ./... -tags=integration -count=1 -race -timeout 5m

lint: ## Run golangci-lint
	golangci-lint run ./...

gosec: ## Run gosec security scanner
	gosec ./...

govulncheck: ## Run Go vulnerability check
	govulncheck ./...

trivy: ## Run Trivy filesystem scan
	trivy fs .

swag: ## Generate Swagger docs
	swag init -g cmd/api/main.go -o api/docs --parseDependency --parseInternal

migrate-up: ## Apply DB migrations (DATABASE_URL required)
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

up: ## Start app + Postgres
	docker compose up --build -d

down: ## Stop app + Postgres
	docker compose down -v --remove-orphans

logs: ## Tail app logs
	docker compose logs -f app

docker-build: ## Build Docker image
	docker build -t $(APP_NAME):local .
