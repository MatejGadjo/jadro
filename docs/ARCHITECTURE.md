# Architecture

Jadro is one Go binary, a static React app, PostgreSQL (with pgvector), Redis and nginx. Everything runs in `docker-compose.yml`.

```
                      ┌──────────────────────────── Go backend (one process) ───────────────────────────┐
  RSS feeds  ───────▶ │ cron: ingest ─▶ normalize ─▶ classify ─▶ store ─▶ embed ─▶ cluster ─▶ rescore     │
                      │                                              (ONNX)   (pgvector)   ─▶ warm cache  │
                      │ cron: cleanup (retention)   cron: daily Web Push                                 │
                      │ HTTP: /api/*  /_prerender/*  /sitemap.xml                                        │
                      └───────────────┬──────────────────────────────┬───────────────────────────────────┘
                                      ▼                              ▼
                               PostgreSQL + pgvector              Redis (cache + rate limits)

  browser ─▶ nginx ─┬─ /api/*, /sitemap.xml ─────────────▶ backend
                    ├─ page routes, crawler user agent ──▶ backend /_prerender/<path>
                    └─ everything else ──────────────────▶ frontend (static SPA)
```

## Ingestion (`backend/internal/ingestion/`, `internal/jobs/`)

Every `INGEST_CRON` tick (default 15 min) the backend fetches every active source in `sources` with a small worker pool and a per-source rate limit. Each item is:

1. **normalized**: title/snippet cleanup, image extraction (`rss/images.go`), canonical domain (`internal/urlutil`)
2. **classified** into a topic (`internal/topics/`). The rules are applied in order: forced source → RSS category map → URL rules → keywords → fallback. The rules are JSON files in `internal/topics/config/`
3. **stored**, deduplicated by URL

`jobs.IngestRunner` guarantees that runs never overlap.

## Embeddings and clustering (`backend/internal/clustering/`)

New articles are queued into a pipeline (`jobs/pipeline.go`) that embeds the normalized title plus a truncated snippet with **paraphrase-multilingual-MiniLM-L12-v2** through ONNX Runtime (384 dimensions). A watchdog detects stalled embeddings.

Clustering is incremental: each new article queries pgvector for the nearest clusters created in the last 24 h (`FindSimilarClusterNeighbors`, ANN top 25), then checks article-level similarity. If the best match is above `SimilarityThreshold` (0.70), the article joins that cluster and the centroid is updated. Otherwise it starts a new cluster. The display title is chosen from the member article closest to the centroid.

## Ranking (`backend/internal/ranking/`)

`ComputeClusterScore` combines √(article count), recency decay (newest article), novelty decay (oldest article), a source-diversity penalty and a velocity boost (articles in the last 6 h). After each ingest the rescore job updates the scores, then the feed cache is rebuilt.

## Serving (`backend/internal/service/`, `internal/rest/`)

- `/api/feed` builds pages of 10 clusters (with their top articles embedded), cached in Redis under a feed version that bumps after every ingest.
- `/api/clusters/:id` returns a cached cluster detail.
- `/_prerender/*` renders Go HTML templates with the same data for search engines and link previews. nginx only sends crawler user agents there.
- The React SPA (`frontend/`) validates every response with zod and uses TanStack Query for caching.

## Abuse protection

| Layer | What it does |
|---|---|
| nginx | Real client IP (trusted hops only), `limit_req` per IP for API, sitemap and crawler pages, `limit_conn`, 16 KiB API bodies |
| Gin middleware | CORS allow-list, 10 s handler timeout, 16 KiB body limit, Redis rate limits per route per IP (fail-closed on writes) |
| Handlers | Bounded query params (`page`, `limit`), push endpoint allow-list (no SSRF), size caps on subscription fields |
| clear-image | Clients only *report* a broken image. The server verifies it with an SSRF-safe probe and caches the verdict |

## Data lifecycle

The cleanup job deletes articles older than `CLEANUP_RETENTION_DAYS` (default 3) every night, then deletes clusters that no longer have any articles. Migrations (`internal/storage/migrations/`, goose) run on startup.
