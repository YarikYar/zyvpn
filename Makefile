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
test: ## Run all tests
	cd backend && go test -v ./...

test-coverage: ## Run tests with coverage
	cd backend && go test -v -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

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
