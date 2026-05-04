.PHONY: help dev up down build migrate migrate-down logs backend miniapp test clean

# Colors
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

help: ## Show this help
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${YELLOW}%-15s${RESET} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
dev: ## Start all services in development mode
	docker-compose up -d postgres redis
	@echo "Waiting for postgres..."
	@sleep 3
	@make migrate
	@echo "Starting backend..."
	cd backend && go run ./cmd/server &
	@echo "Starting miniapp..."
	cd miniapp && npm run dev

up: ## Start all services with docker-compose
	docker-compose up -d

down: ## Stop all services
	docker-compose down

build: ## Build all services
	docker-compose build

logs: ## Show logs
	docker-compose logs -f

# Backend
backend: ## Run backend only
	cd backend && go run ./cmd/server

backend-build: ## Build backend binary
	cd backend && go build -o bin/server ./cmd/server

# Mini App
miniapp: ## Run miniapp development server
	cd miniapp && npm run dev

miniapp-build: ## Build miniapp for production
	cd miniapp && npm run build

miniapp-install: ## Install miniapp dependencies
	cd miniapp && npm install

# Database
migrate: ## Run database migrations
	cd backend && go run -tags migrate ./cmd/migrate up

migrate-down: ## Rollback database migrations
	cd backend && go run -tags migrate ./cmd/migrate down

migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	cd backend && migrate create -ext sql -dir migrations -seq $(name)

db-shell: ## Open PostgreSQL shell
	docker-compose exec postgres psql -U zyvpn -d zyvpn

db-reset: ## Reset database (drop and recreate)
	docker-compose exec postgres psql -U zyvpn -c "DROP DATABASE IF EXISTS zyvpn;"
	docker-compose exec postgres psql -U zyvpn -c "CREATE DATABASE zyvpn;"
	@make migrate

# Redis
redis-shell: ## Open Redis CLI
	docker-compose exec redis redis-cli

# Testing
test: test-unit ## Run unit tests (default — no external deps)

test-unit: ## Run unit tests only (no postgres / no live XUI)
	cd backend && go test -count=1 -v ./...

test-coverage: ## Run unit tests with coverage report
	cd backend && go test -count=1 -v -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# --- Integration tests (require external services) ---
#
# Subscription/DB integration: requires a postgres instance.
#   TEST_DATABASE_URL — full DSN. Example:
#     postgres://zyvpn:zyvpn@localhost:55434/zyvpn_test?sslmode=disable
# The test will run migrations and TRUNCATE tables between tests, so point
# it at a throwaway database.
test-integration-db: ## Run subscription/DB integration tests (needs TEST_DATABASE_URL)
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "TEST_DATABASE_URL not set. Example:"; \
		echo "  TEST_DATABASE_URL=postgres://zyvpn:zyvpn@localhost:55434/zyvpn_test?sslmode=disable make test-integration-db"; \
		exit 1; \
	fi
	cd backend && go test -tags=integration -count=1 -v ./internal/service/...

# Helpers to spin a throwaway postgres for integration tests.
test-pg-up: ## Start a throwaway postgres on :55434 for integration tests
	docker run -d --rm --name zyvpn-pg-test \
		-e POSTGRES_USER=zyvpn -e POSTGRES_PASSWORD=zyvpn -e POSTGRES_DB=zyvpn_test \
		-p 55434:5432 postgres:16-alpine
	@echo "Waiting for postgres..."
	@until docker exec zyvpn-pg-test pg_isready -U zyvpn >/dev/null 2>&1; do sleep 1; done
	@echo "Ready. Export:"
	@echo "  export TEST_DATABASE_URL=postgres://zyvpn:zyvpn@localhost:55434/zyvpn_test?sslmode=disable"

test-pg-down: ## Stop the throwaway postgres
	-docker stop zyvpn-pg-test

# Live XUI integration: requires real 3x-ui panel credentials. Use a
# non-prod inbound — the tests create and delete a test client.
#   XUI_TEST_BASE_URL
#   XUI_TEST_USERNAME
#   XUI_TEST_PASSWORD
#   XUI_TEST_INBOUND_ID
test-integration-xui: ## Run XUI live integration tests (needs XUI_TEST_* env)
	@if [ -z "$$XUI_TEST_BASE_URL" ]; then \
		echo "XUI_TEST_BASE_URL/USERNAME/PASSWORD/INBOUND_ID not set."; \
		exit 1; \
	fi
	cd backend && go test -tags=integration -count=1 -v ./internal/xui/...

test-integration: test-integration-db test-integration-xui ## Run all integration tests

# Linting
lint: ## Run linters
	cd backend && golangci-lint run
	cd miniapp && npm run lint

# Cleaning
clean: ## Clean build artifacts
	cd backend && rm -rf bin/
	cd miniapp && rm -rf dist/ node_modules/
	docker-compose down -v

# Dependencies
deps: ## Install all dependencies
	cd backend && go mod download
	cd miniapp && npm install

# Docker
docker-build-backend: ## Build backend Docker image
	docker build -t zyvpn-backend ./backend

docker-build-miniapp: ## Build miniapp Docker image
	docker build -t zyvpn-miniapp ./miniapp

# Production
deploy: ## Build and deploy all services
	docker-compose build
	docker-compose up -d

restart: ## Restart all services
	docker-compose restart

rebuild: ## Rebuild and restart specific service (usage: make rebuild s=backend)
	docker-compose build $(s)
	docker-compose up -d $(s)

status: ## Show status of all services
	docker-compose ps
