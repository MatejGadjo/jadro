# News Aggregator Backend

Go backend for a Macedonian-language news aggregator. Ingests RSS feeds, clusters articles into stories, scores and serves them via a REST API.

## Tech Stack

| Layer | Library |
|-------|---------|
| HTTP | Gin |
| DI / lifecycle | uber/fx |
| Logging | uber/zap |
| Database | pgx/v5 + pgvector (vector(384)) |
| Cache | go-redis/v9 |
| RSS | gofeed + go-resty |
| Worker pool | alitto/pond/v2 |
| Embeddings | yalue/onnxruntime_go + knights-analytics/hugot |
| Scheduler | robfig/cron/v3 |
| Config | joho/godotenv |

## Quick Start

```bash
# 1. Copy and fill in environment variables
cp .env.example .env

# 2. Create the database and enable pgvector
psql -c "CREATE DATABASE newsaggregator;"
psql -d newsaggregator -c "CREATE EXTENSION IF NOT EXISTS vector;"

# 3. Run migrations
migrate -path migrations -database "$DATABASE_URL" up

# 4. (Optional) Download ONNX model files — see ML Models section below

# 5. Start the server
make run
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | Postgres DSN (`postgres://user:pass@host/db?sslmode=disable`) |
| `REDIS_URL` | — | — | Redis URL (`redis://localhost:6379`). Empty = caching disabled |
| `HTTP_ADDR` | — | `:8080` | Listen address |
| `HTTP_SHUTDOWN_TIMEOUT` | — | `5s` | Graceful shutdown timeout |
| `INGEST_CRON` | — | `*/15 * * * *` | Cron expression for RSS ingestion |
| `APP_ENV` | — | `development` | Environment name |
| `EMBEDDING_WORKERS` | — | `4` | Parallel workers for embedding generation |
| `ONNX_RUNTIME_LIB_PATH` | — | — | Path to `libonnxruntime.so`. Empty = embeddings disabled |
| `ONNX_MODEL_PATH` | — | `./models/.../model.onnx` | Path to ONNX model file |
| `ONNX_TOKENIZER_PATH` | — | `./models/.../tokenizer.json` | Path to tokenizer config |
| `DB_MAX_CONNS` | — | pgx default | Max DB pool connections |
| `DB_MIN_CONNS` | — | pgx default | Min DB pool connections |
| `ALLOWED_ORIGINS` | — | `*` | Comma-separated allowed CORS origins. **Set this in production** (`https://yourapp.com`) |

## ML Models

The ONNX runtime and model files are large binaries — **do not commit to git**.

Model used: `paraphrase-multilingual-MiniLM-L12-v2` (384-dimensional embeddings, supports Macedonian).

| File | Size | Notes |
|------|------|-------|
| `libonnxruntime.so` | ~200 MB | ONNX Runtime shared library |
| `model.onnx` | ~120 MB | Sentence embedding model |
| `tokenizer.json` | <1 MB | Tokenizer config |

If `ONNX_RUNTIME_LIB_PATH` is unset, the app starts without embeddings — clustering falls back to rule-based grouping. This is the intended behaviour for local dev without the model files.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — DB ping, last ingest time, embedding status |
| `GET` | `/api/feed` | Main feed. Params: `topic` (optional), `page` (optional, default 0) |
| `GET` | `/api/clusters/:id` | Cluster detail with articles and sources |
| `GET` | `/api/articles/latest` | Latest articles. Params: `limit` (optional, default 8) |
| `PATCH` | `/api/articles/:id/clear-image` | Clear a broken article image |

Feed response shape:
```json
{
  "clusters": [...],
  "featured_clusters": [...],
  "top_clusters_by_category": [...]
}
```

## Architecture

```
RSS feeds
    │
    ▼
Ingestion (rss.StartRSSIngestion)
    │  fetch + normalize + classify topic
    ▼
Storage (postgres)
    │  article URL dedup via DB constraint
    ▼
Embedding generation (onnxruntime, worker pool)
    │  384-dim vectors stored via pgvector
    ▼
Clustering (incremental cosine similarity)
    │  articles grouped into story clusters
    ▼
Scoring / Rescore (ranking)
    │  clusters scored by recency + source diversity
    ▼
Redis cache (feed:home:p{n}, feed:topic:{X}:p{n})
    │  invalidated after each ingest run
    ▼
REST API (Gin)
```

The scheduler (`robfig/cron`) runs ingestion every `INGEST_CRON` interval. On each run: ingest → rescore → invalidate Redis feed cache.

## Project Structure

```
cmd/api/          — entry point (fx app wiring)
internal/
  app/            — fx module (dependency wiring)
  cache/          — Redis client (Get, Set, DeleteByPattern)
  clustering/     — incremental story clustering + embedding generation
  config/         — env var loading
  http/           — Gin server, routes, middleware, handlers
  ingestion/rss/  — RSS fetch + article normalization
  jobs/           — scheduler, pipeline dispatcher, rescore job
  logger/         — zap setup
  ranking/        — cluster score computation
  storage/        — domain models
  storage/postgres/ — all SQL queries (pgx)
  topics/         — topic classification
migrations/       — SQL migrations (numbered)
models/           — ONNX model files (gitignored)
```

## Commands

```bash
make run            # run the API (requires DATABASE_URL)
make build          # compile to bin/newsaggregator
make test           # run all tests
make test-coverage  # run tests + generate coverage.html
make tidy           # tidy go.mod/go.sum
```

## Security

### Implemented
- **Rate limiting** — Redis-backed per-IP limits: `GET /api/feed` 30 req/min, `GET /api/clusters/:id` and `GET /api/articles/latest` 60 req/min, `PATCH /api/articles/:id/clear-image` 20 req/min. No-op when `REDIS_URL` is unset.
- **CORS** — controlled via `ALLOWED_ORIGINS` env var. Defaults to `*` (wildcard) if unset — **must be set in production**.
- **Request timeout** — 10 s context deadline on all handlers. DB and Redis calls cancel automatically on breach.

### Before deploying to production

| Item | Action |
|------|--------|
| Trusted proxies | Call `r.SetTrustedProxies([]string{"127.0.0.1"})` (nginx) or Cloudflare IP ranges in `internal/http/server.go`. Current default (`nil`) uses `RemoteAddr` directly — correct for bare VPS, wrong behind a proxy |
| `PATCH /api/articles/:id/clear-image` | Unauthenticated write endpoint — move under `/internal/*` with bearer token or restrict to trusted IPs before going public |
| `/health` | Exposes `last_ingest_at` (scheduler timing). No rate limit. Consider restricting or removing from public surface |
| `ALLOWED_ORIGINS` | Set to your frontend domain — do not leave as wildcard in production |

## Scope

- Macedonian language, RSS sources only
- No authentication or user accounts
- No personalization or per-user state
- Feed is cluster-based (articles only available on cluster detail)
