# Jadro — Macedonian News Aggregator

Clusters Macedonian news from multiple RSS sources into story groups. Each story surfaces articles from different outlets so you can compare perspectives.

**Stack:** Go + Gin · PostgreSQL + pgvector · Redis · React + Vite · nginx · Docker

---

## Architecture

```
Cloudflare → nginx (TLS, static files) → Go/Gin :8080 → PostgreSQL + pgvector
                                                       → Redis (feed cache)
```

- **Ingestion** — cron-driven RSS fetch every 15 min, topic classification, dedup by URL
- **Clustering** — ONNX embedding model (multilingual MiniLM-L12-v2, 384-dim) + pgvector cosine similarity; falls back to rule-based clustering when model files are absent
- **Feed** — scored clusters served via `/api/feed`; Redis read-through cache with 5 min TTL, invalidated after every ingest
- **Frontend** — React SPA, TanStack Router + Query, Tailwind + shadcn/ui

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose v2
- Make (optional, for backend dev commands)
- Node 20+ (frontend dev only)
- Go 1.25+ (backend dev only)

ML model files are required for semantic clustering (see [ML Models](#ml-models)). Without them the app starts and uses rule-based clustering.

---

## Quick Start

```bash
git clone https://github.com/MatejGadjo/NewsAggregator
cd NewsAggregator

# Set required env vars
echo "POSTGRES_PASSWORD=changeme" > .env

# Start everything
docker compose up --build
```

App is available at `http://localhost` (nginx). Backend health: `http://localhost/api/health`.

> The backend triggers an RSS ingest run on startup and every 15 minutes after.

---

## Environment Variables

### Backend

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres DSN, e.g. `postgres://user:pass@host:5432/db?sslmode=disable` |
| `APP_ENV` | No | `development` | Set to `production` to enable JSON logging and enforce required secrets |
| `HTTP_ADDR` | No | `:8080` | Listen address |
| `LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `REDIS_URL` | No | — | e.g. `redis://localhost:6379`; empty disables feed caching |
| `ALLOWED_ORIGINS` | No | `*` | Comma-separated CORS origins, e.g. `https://jadro.mk,https://www.jadro.mk` |
| `INGEST_CRON` | No | `*/15 * * * *` | Cron expression for RSS ingestion |
| `CLEANUP_CRON` | No | `0 3 * * *` | Cron expression for old-article cleanup |
| `CLEANUP_RETENTION_DAYS` | No | `3` | Articles older than this are deleted |
| `ONNX_RUNTIME_LIB_PATH` | No | — | Path to `libonnxruntime.so`; empty disables ML clustering |
| `ONNX_MODEL_PATH` | No | `./models/.../model.onnx` | Path to `model.onnx` |
| `ONNX_TOKENIZER_PATH` | No | `./models/.../tokenizer.json` | Path to `tokenizer.json` |

### Frontend

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VITE_API_BASE_URL` | No | `http://localhost:8080` | Backend base URL |
| `VITE_GA_MEASUREMENT_ID` | No | — | GA4 measurement ID (`G-XXXXXXXXXX`); empty disables analytics |

---

## ML Models

Semantic clustering requires three files that are **not in git** (total ~320 MB):

| File | Size | Location inside container |
|------|------|---------------------------|
| `libonnxruntime.so` | ~200 MB | `backend/models/onnxruntime-linux-x64-<ver>/lib/` |
| `model.onnx` | ~120 MB | `backend/models/paraphrase-multilingual-MiniLM-L12-v2/` |
| `tokenizer.json` | <1 MB | `backend/models/paraphrase-multilingual-MiniLM-L12-v2/` |

**Download steps:**

```bash
# 1. ONNX Runtime shared library (adjust version as needed)
wget https://github.com/microsoft/onnxruntime/releases/download/v1.23.2/onnxruntime-linux-x64-1.23.2.tgz
tar -xzf onnxruntime-linux-x64-1.23.2.tgz -C backend/models/

# 2. Model + tokenizer (Hugging Face)
mkdir -p backend/models/paraphrase-multilingual-MiniLM-L12-v2
cd backend/models/paraphrase-multilingual-MiniLM-L12-v2

wget https://huggingface.co/sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2/resolve/main/onnx/model.onnx
wget https://huggingface.co/sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2/resolve/main/tokenizer.json
```

`docker-compose.yml` volume-mounts `./backend/models` → `/app/models` inside the container.

Without these files the backend logs a warning and falls back to rule-based clustering. Everything else works normally.

---

## API Reference

All routes are under `/api`. No versioning prefix.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/health` | None | Health check — DB, Redis, counts, last ingest time |
| `GET` | `/api/feed` | None | Paginated cluster feed. Query: `topic=`, `page=` |
| `GET` | `/api/clusters/:id` | None | Cluster detail with all articles |
| `GET` | `/api/articles/latest` | None | Latest articles |
| `PATCH` | `/api/articles/:id/clear-image` | None | Remove broken image URL |

**Rate limits** (Redis-backed, fails open):
- `/api/feed` — 30 req/min per IP
- `/api/clusters/:id`, `/api/articles/latest` — 60 req/min per IP
- `/api/articles/:id/clear-image` — 20 req/min per IP

Ingestion and cleanup run on an in-process cron (`INGEST_CRON` / `CLEANUP_CRON`); there are no manual trigger endpoints.

---

## Development

### Backend

```bash
cd backend
cp .env.example .env   # fill in DATABASE_URL at minimum
make run               # starts on :8080
make test              # run tests
make build             # compile to bin/newsaggregator
```

### Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev            # Vite dev server on :5173
npm run build          # type-check + production build
npm run lint
```

---

## Deployment

See [PRODUCTION.md](PRODUCTION.md) for the full VPS deployment guide including nginx config, TLS setup, GitHub Actions secrets, and a post-deploy checklist.
