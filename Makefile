.DEFAULT_GOAL := help
.PHONY: help setup models up down logs dev-deps dev-backend dev-frontend test lint vapid

COMPOSE := docker compose

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Create .env files from the examples (never overwrites)
	@for f in .env backend/.env frontend/.env; do \
		if [ -f $$f ]; then echo "✓ $$f exists"; else cp $$f.example $$f && echo "+ created $$f"; fi; \
	done

models: ## Download + verify the embedding model and ONNX Runtime (~480 MB)
	@./scripts/download-models.sh

up: ## Build and start the full stack on http://localhost:8000
	$(COMPOSE) up -d --build
	@echo "Jadro is starting on http://localhost:8000 — the first ingest runs within 15 minutes (see README)."

down: ## Stop the stack (data is kept in the postgres volume)
	$(COMPOSE) down

logs: ## Follow backend logs
	$(COMPOSE) logs -f backend

dev-deps: ## Start only Postgres + Redis for local development
	$(COMPOSE) up -d postgres redis

dev-backend: ## Run the Go backend on :8080 (needs dev-deps and backend/.env)
	cd backend && go run ./cmd/api

dev-frontend: ## Run the Vite dev server on :5173 (needs frontend/.env)
	cd frontend && npm install && npm run dev

test: ## Run backend and frontend tests
	cd backend && go test -race ./...
	cd frontend && npm test

lint: ## Run golangci-lint, ESLint and the TypeScript checker
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint && npm run typecheck

vapid: ## Generate a VAPID keypair for Web Push
	cd backend && go run ./cmd/vapidgen
