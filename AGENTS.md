# Jadro: guide for AI coding agents

Jadro is a news aggregator: RSS → normalize/classify → Postgres → ONNX embeddings → pgvector clustering → ranking → Go API → React SPA. It is a monorepo:

- `backend/`: Go API and in-process jobs. See [backend/AGENTS.md](backend/AGENTS.md).
- `frontend/`: React + Vite SPA. See [frontend/AGENTS.md](frontend/AGENTS.md).
- `nginx/nginx.conf`: edge proxy: rate limits, real client IP, crawler → prerender routing.
- `docker-compose.yml`: the full stack; `scripts/download-models.sh` fetches the ML model.

## Commands (from the repo root)

- `make up` / `make down` / `make logs`: full stack on http://localhost:8000
- `make test`: `go test -race ./...` + `vitest run`
- `make lint`: golangci-lint + eslint + `tsc`

Run `make lint test` before declaring a change done.

## Ground rules

- The app is deployed in production. Keep changes backward compatible with existing env vars and data. New env vars need safe defaults.
- Never commit secrets, `.env` files, model binaries or personal tool settings.
- Never edit generated sqlc code or already-shipped migrations (see CONTRIBUTING.md).
- Public endpoints are unauthenticated: validate and bound every input, and never trust client claims for writes.
- No user accounts, personalization or tracking by default. Don't add these without explicit instruction.
