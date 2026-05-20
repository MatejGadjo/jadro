# News Aggregator Backend (Project Instructions)

## Tech stack
- **Language**: Go 1.25
- **HTTP**: Gin
- **DI**: uber/fx
- **Logging**: uber/zap
- **Database**: pgx/v5 (Postgres), pgvector-go (vector(384)), go-redis/v9 (Redis)
- **RSS**: gofeed + go-resty (HTTP client, 10s timeout)
- **Worker pool**: alitto/pond/v2
- **Embeddings**: yalue/onnxruntime_go + knights-analytics/hugot, daulet/tokenizers
- **Testing**: testify
- **Config**: godotenv

## Scope and goals
- MVP scope: Macedonian language, RSS feeds only, no personalization or accounts.
- Primary flow: ingest RSS -> normalize -> classify topic -> store -> generate embeddings -> cluster -> score -> serve feeds.

## Commands
- `make run` — run the API (requires `DATABASE_URL`)
- `make build` — compile to `bin/newsaggregator`
- `make test` — run all tests
- `make test-coverage` — run tests + generate `coverage.html`
- `make tidy` — tidy go.mod/go.sum

## How to run
- Requires `DATABASE_URL` env var at minimum.
- ONNX embeddings are optional; if unconfigured or model files are missing, embeddings are disabled.
  - `ONNX_RUNTIME_LIB_PATH` (libonnxruntime), `ONNX_MODEL_PATH`, `ONNX_TOKENIZER_PATH`.

## Architectural notes
- DI via `fx` in `cmd/api/main.go`.
- `cmd/api/main.go` currently triggers RSS ingestion and clustering on startup (manual testing flow).
- Postgres stores articles/clusters; embeddings stored via `pgvector` (vector size 384).

## API contract
- `GET /api/feed?topic=&page=` → `{ clusters, featured_clusters, top_clusters_by_category }`
- `GET /api/clusters/:id` → `{ cluster, articles[] }` (articles include nested `source`)
- Listens on `:8080` by default.
- No versioning prefix; all routes are `/api/...`.

## Data and invariants
- Article URL is unique (dedup relies on DB constraint).
- Embedding dimension is 384 across storage and ONNX model.
- Feeds are served via `/api/feed` using cluster scores.

## Intentionally absent
- No authentication or authorization middleware.
- No multi-tenancy or user accounts.
- No personalization or per-user state.
- No article-level feed endpoint (feed is cluster-based only).
- Do not add any of the above without explicit instruction.

## Where to look
- Ingestion: `internal/ingestion`
- Clustering: `internal/clustering`
- Ranking: `internal/ranking`
- Storage/queries: `internal/storage` and `internal/storage/postgres`
- HTTP: `internal/http`
