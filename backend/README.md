# Jadro backend

Go service for [Jadro](../README.md). A single binary serves the REST API and runs the ingest, cleanup and push jobs on an in-process cron.

```bash
cp .env.example .env    # points at Postgres/Redis from `make dev-deps`
make run                # API on :8080 (migrations run automatically)
make test               # go test ./...
make vapid              # generate Web Push keys
```

- Package layout, env vars and the API contract: [AGENTS.md](AGENTS.md)
- SQL/sqlc and migration workflow: [CONTRIBUTING.md](../CONTRIBUTING.md#backend-notes)
- How ingestion, clustering and ranking fit together: [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)

The embedding model is required for clustering. Download it with `make models` from the repo root.
