# Production Readiness Roadmap

Status tracker for getting NewsAggregator from MVP to production-ready deployment.

---

## 1. Deployment Architecture

### Current State
- Go backend + React frontend as separate processes
- PostgreSQL with pgvector extension
- Redis configured but unused
- ONNX model files loaded from local filesystem on startup
- No containerization / orchestration defined beyond `docker-compose.yml` (dev only)

### Recommended Stack
```
[Cloudflare / CDN]
        ↓
[nginx reverse proxy] — TLS termination, static asset serving, rate limiting
    ├── /             → Frontend static files (Vite build output)
    └── /api/*        → Go backend (:8080)
        ↓
[Go backend] — Gin + onnxruntime
        ├── PostgreSQL + pgvector
        └── Redis
```

### ML Model Deployment
The ONNX runtime (`libonnxruntime.so`) and model files are large binary artifacts that need special handling:

| File | Size (approx) | Strategy |
|------|--------------|---------|
| `libonnxruntime.so` | ~200 MB | Bake into Docker image or mount as volume |
| `model.onnx` | ~120 MB | Same — do NOT store in git |
| `tokenizer.json` | <1 MB | Can be in git or downloaded at build time |

**Options:**
- **Docker image** (simplest): Copy model files into the image at build time. Image will be ~400 MB+.
- **Object storage mount** (scalable): Store in S3/Backblaze/Hetzner Object Storage, download on container start.
- **Separate model server** (advanced): Run a dedicated embedding service (e.g., Python FastAPI with sentence-transformers) and call it over HTTP. Decouples ML scaling from API scaling.

**Graceful degradation is already implemented** — if ONNX env vars are unset, embeddings are disabled and clustering falls back to rule-based. Use this as a circuit breaker in production if the model server is unavailable.

### Checklist
- [x] Write `Dockerfile` for backend (multi-stage: builder + runtime with model files)
- [x] Write `Dockerfile` for frontend (Vite build → nginx static server)
- [x] Write production `docker-compose.yml`
- [x] Decide: volume mount (models stay on VPS, image stays lean)
- [x] Set `ONNX_RUNTIME_LIB_PATH`, `ONNX_MODEL_PATH`, `ONNX_TOKENIZER_PATH` in production env (hardcoded in compose, volume path is fixed)
- [x] `CREATE EXTENSION IF NOT EXISTS vector` handled automatically by goose migration 00002
- [x] Migrations auto-run on startup via goose + `//go:embed` (7 consolidated migrations)
- [x] Create `.env.example` at project root
- [ ] **Decide Redis hosting**: self-hosted on VPS (free, single point of failure) vs. Upstash free tier (10k cmds/day, 256 MB, survives VPS restart). Either works at current scale — decide at deploy time.

---

## 2. Scheduled Article Fetching

### Implemented
`robfig/cron/v3` in-process scheduler — single binary, no external cron daemon needed. Fires immediately on startup then repeats on a cron expression. `atomic.Bool` overlap guard prevents concurrent runs.

- `internal/jobs/scheduler.go` — `Scheduler` struct with `IngestFunc` type (breaks `jobs`↔`rss` import cycle), cron instance, overlap guard
- `internal/config/config.go` — `IngestCron` field, `INGEST_CRON` env var (default `*/15 * * * *`)
- `internal/app/modules.go` — scheduler wired into fx lifecycle, rescore runs after every ingest
- `cmd/api/main.go` — manual startup goroutine removed

### Checklist
- [x] Implement `internal/jobs/scheduler.go` with robfig/cron v3
- [x] Add `INGEST_CRON` env var (default: `*/15 * * * *`)
- [x] Log ingestion run start/end
- [x] Overlap guard — skips run if previous still in progress
- [x] Rescore runs after every ingest
- [x] Log article count per run (`StartRSSIngestion` now returns `(int, error)`, logged as `new_articles`)
- [x] Expose `last_ingest_at` in `/health` response (populated after first completed run)

---

## 3. Caching Strategy

### Current State ✅
Read-through caching implemented on `GET /api/feed`. Redis client wired via fx. Post-ingest invalidation hook fires after every successful ingest+rescore cycle.

### What to Cache

| Endpoint | TTL | Cache Key | Invalidation |
|----------|-----|-----------|-------------|
| `GET /api/feed` | 5 min | `feed:home:p{n}` | On ingestion job completion (invalidate all `feed:*` keys) |
| `GET /api/feed?topic=X` | 5 min | `feed:topic:{X}:p{n}` | On ingestion job completion |

> `GET /api/stats` is currently not cached and of questionable value in production — see **Known Issues** note below.

### Invalidation Strategy
**Post-ingest hook (Option A):** Scheduler calls `cache.DeleteByPattern("feed:*")` after each successful ingest run. This ensures data is always fresh after ingestion completes, with no stale window beyond the ingest interval. Pure TTL fallback (5 min) acts as a safety net in case the hook fails.

### Known Issues / Concerns

#### Stats endpoint (`GET /api/stats`)
The stats endpoint serves the "Денес" sidebar widget and hits Postgres on every request. It is currently not cached, not invalidated on ingest, and its usefulness in production is unclear (the sidebar data may not justify a live DB call on every page load). **Decision needed:** remove the endpoint, cache it, or drive the widget from the feed response instead.

### Implementation Plan
The pattern is: read-through cache in `FeedHandler`, invalidate via a hook called at end of each ingestion run. `internal/cache/redis.go` needs to be built out (currently just a blank import).

```go
// In FeedHandler.GetFeed
key := fmt.Sprintf("feed:home:p%d", page)  // or feed:topic:{X}:p{n}
if raw, err := h.cache.Get(ctx, key); err == nil {
    c.Data(http.StatusOK, "application/json", raw)
    return
}
// ... build response ...
h.cache.Set(ctx, key, jsonBytes, 5*time.Minute)
```

### Checklist
- [x] Build out `internal/cache/redis.go` — `Get`, `Set`, `DeleteByPattern`, real client wired via fx
- [x] Add `REDIS_URL` env var (empty = caching disabled; set to `redis://localhost:6379` locally)
- [x] Inject `*cache.Client` into `FeedHandler`
- [x] Implement read-through caching in `GET /api/feed` (homepage + topic variants, 5 min TTL)
- [x] Add post-ingest invalidation hook in `Scheduler` — calls `cache.DeleteByPattern("feed:*")` on run completion
- [x] Stats endpoint removed — `GET /api/stats`, `StatsHandler`, `DashboardStats`, `useStats` hook, and sidebar widget all deleted
- [ ] Consider Cloudflare edge caching for public endpoints (set `Cache-Control: public, max-age=300`)

---

## 4. Atomic Feed Updates (Homepage State)

### Problem
The frontend currently uses TanStack Query with `staleTime: 5min`. When React Query refetches, it replaces data immediately as the response arrives — this means the UI can show a mix of old clusters disappearing and new ones appearing as React reconciles the state change. In practice, the clustering backend returns a fully computed response in one shot, so the issue is less about partial updates and more about **visual jank** when the feed list re-renders and cluster cards shift positions.

### Solution: Snapshot-based feed versioning
Backend returns a `feed_version` (timestamp or incrementing integer) with each feed response. Frontend only commits the new feed to display state when the version changes — preventing stale re-renders and ensuring the full new batch is shown at once.

**Backend change** — add `feed_version` to `FeedResponse`:
```go
type FeedResponse struct {
    FeedVersion     int64     `json:"feed_version"`   // unix timestamp of last ingest run
    Clusters        []Cluster `json:"clusters"`
    FeaturedClusters []Cluster `json:"featured_clusters"`
    TopClustersByCategory map[string][]Cluster `json:"top_clusters_by_category"`
}
```

**Frontend change** — compare versions before committing update:
```ts
// In useFeed hook — keep previous data until version actually changes
const { data, previousData } = useQuery({
  queryKey: ['feed', topic, page],
  queryFn: fetchFeed,
  placeholderData: keepPreviousData,  // TanStack Query v5 pattern
  staleTime: 5 * 60 * 1000,
})
// Show placeholderData (old feed) until new feed_version arrives
```

`keepPreviousData` (now `placeholderData: keepPreviousData` in TanStack Query v5) already prevents the "blank flash" between paginations. The key improvement is not triggering UI re-renders on refetch unless the feed_version actually changed.

### Checklist
- [x] Rescore writes all cluster scores atomically via a single `unnest` bulk UPDATE — no more per-cluster row-by-row updates mid-sort
- [x] Post-ingest hook writes `meta:feed_version` (unix timestamp) to Redis after each successful rescore
- [x] `feed_version` included in every `/api/feed` response (baked into cached JSON, refreshed on cache invalidation)
- [x] Frontend: `placeholderData: keepPreviousData` confirmed set in `useFeed`
- [x] Frontend: `select` ref-check in `useFeed` — returns previous data reference when `feed_version` unchanged, preventing re-render on stale refetches between ingest cycles
- [ ] Optionally: add a subtle "X new stories available — refresh" banner rather than auto-updating

---

## 5. Security & Rate Limiting

### Current State ✅ (partial)
- CORS: reads from `ALLOWED_ORIGINS` env var (empty = wildcard for local dev)
- Redis-backed rate limiting on all `/api/*` endpoints
- 10s request timeout on all handlers
- No authentication (intentional — public feed)
- DB connection string passed via env var (acceptable if secrets are managed properly)

### Implemented

#### 5.1 Rate Limiting
`go-redis/redis_rate/v10` (GCRA algorithm) — per-IP, Redis-backed, fails open on Redis error.
No-op when `REDIS_URL` is unset (Redis disabled).

| Endpoint | Limit |
|----------|-------|
| `GET /api/feed` | 30 req/min per IP |
| `GET /api/clusters/:id` | 60 req/min per IP |
| `GET /api/articles/latest` | 60 req/min per IP |

Rate limit headers returned on every request: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After` (on 429).

- `internal/http/middleware/ratelimit.go` — `RateLimiter` struct, `Limit(redis_rate.Limit)` middleware factory
- `internal/http/routes.go` — per-route limits applied

#### 5.2 CORS
`ALLOWED_ORIGINS` env var — comma-separated list of allowed origins. Empty = wildcard (`*`).
Set to your production domain before deploy: `ALLOWED_ORIGINS=https://yourapp.com,https://www.yourapp.com`

#### 5.3 Request Timeout
10s context deadline on all handlers via `internal/http/middleware/timeout.go`. DB and Redis calls
respect context cancellation — they abort naturally when the deadline is exceeded.

#### 5.4 Infrastructure Layer
Many of these are better handled at the reverse proxy / CDN level:
- **Cloudflare**: Free tier provides DDoS protection, bot detection, and edge rate limiting
- **nginx `limit_req_zone`**: Rate limit by IP before hitting the Go process
- **fail2ban**: Block repeat offenders at firewall level

**Recommended hosting:** Hetzner CX22 (€4/mo) + Cloudflare free tier. Cloudflare is the first wall;
Go middleware is defense-in-depth.

#### 5.5 Secrets Management
- Never commit `.env` files — use a secrets manager in production (Doppler, HashiCorp Vault, AWS Secrets Manager, or Hetzner/environment variables injected by orchestrator)
- Rotate `INTERNAL_JOB_SECRET` periodically

### Checklist
- [x] Implement Redis-backed rate limiting middleware in `internal/http/middleware/`
- [x] Restrict CORS to known production domains (`ALLOWED_ORIGINS` env var)
- [x] Add request timeout middleware (10s, context-based)
- [x] Add `ALLOWED_ORIGINS` env var (comma-separated, empty = wildcard)
- [ ] Protect `/internal/*` endpoints with IP allowlist or bearer token (no `/internal/*` routes exist yet)
- [ ] Put the app behind Cloudflare (free plan) or nginx with `limit_req`
- [ ] Run `security-review` skill after implementing security changes

---

## 6. Monitoring & Observability

### Structured Logging
zap is already in place. Ensure log levels are configurable via `LOG_LEVEL` env var and that production logs are in JSON format for log aggregation (Loki, Datadog, etc.).

### Metrics
Add Prometheus metrics endpoint (`/metrics`) using `prometheus/client_golang`:
- Request count and latency per endpoint (histogram)
- Active DB connections
- RSS ingestion duration and article count per run
- Embedding generation latency
- Cache hit/miss ratio
- Cluster count and article count (gauges)

### Health Check Improvements
Current `/health` returns DB ping only. Extend:
```json
{
  "ok": true,
  "db": "up",
  "redis": "up",
  "last_ingest_at": "2026-05-18T14:23:00Z",
  "article_count": 12400,
  "cluster_count": 843,
  "embedding_enabled": true
}
```

### Alerts
- Ingestion run has not completed in > 30 minutes → PagerDuty / alertmanager
- Error rate on `/api/feed` > 1% → alert
- DB connection pool exhaustion → alert
- Embedding failure rate > 10% in a run → alert

### Checklist
- [ ] Add `LOG_LEVEL` env var, ensure production uses JSON output
- [ ] Add Prometheus `/metrics` endpoint
- [x] Extend `/health` with Redis status, last ingest time, embedding enabled flag
- [ ] Set up Grafana + Prometheus (or hosted equivalent) for dashboards
- [ ] Configure alerting on ingestion failures

---

## 7. Infrastructure & CI/CD

### Recommended Hosting (cost-efficient)
| Component | Option A (simple) | Option B (scalable) |
|-----------|------------------|---------------------|
| App server | Hetzner CX22 (€4/mo) | Fly.io or Railway |
| Database | Managed PostgreSQL (Hetzner, Neon, Supabase) | Same |
| Redis | Same server or Upstash Redis | Upstash Redis (serverless) |
| CDN / DDoS | Cloudflare free tier | Same |
| Static files | nginx on same server | Cloudflare Pages |

**Minimum viable server spec**: 2 vCPU, 4 GB RAM (ONNX runtime needs ~500 MB on load, PostgreSQL ~1 GB, Go process ~100 MB).

### CI/CD
```yaml
# .github/workflows/deploy.yml (sketch)
on: push to main
jobs:
  test: go test ./... && npm run build
  build: docker build + push to registry
  deploy: ssh deploy command or kubectl rollout restart
```

### Checklist
- [ ] Write `Dockerfile` (backend, multi-stage)
- [ ] Write `Dockerfile` (frontend, nginx static)
- [ ] Write production `docker-compose.yml`
- [ ] Write `.github/workflows/test.yml` (run `go test ./...` + `npm run build` on PR)
- [ ] Write `.github/workflows/deploy.yml` (deploy on merge to main)
- [ ] Create `.env.example` for backend and frontend
- [ ] Document model file download steps in deployment guide

---

## 8. Database Cleanup Job

### Problem
Articles and clusters accumulate indefinitely. At 15-min ingest intervals with ~50 sources, the `articles` and `clusters` tables will grow to tens of thousands of rows within weeks. No automatic pruning = unbounded disk usage.

### Strategy
Scheduled cleanup job (runs alongside the ingest scheduler) that hard-deletes rows older than **3 days** in dependency order: `cluster_articles` join table → `articles` → `clusters` (with no remaining articles).

### Deletion Order
```sql
-- 1. Remove join table rows for old articles
DELETE FROM cluster_articles
WHERE article_id IN (
    SELECT id FROM articles WHERE published_at < NOW() - INTERVAL '3 days'
);

-- 2. Delete old articles
DELETE FROM articles WHERE published_at < NOW() - INTERVAL '3 days';

-- 3. Delete clusters that now have no articles
DELETE FROM clusters
WHERE id NOT IN (SELECT DISTINCT cluster_id FROM cluster_articles);
```

### Implementation Plan
- Add `CleanupJob` to `internal/jobs/` — reuses the same `robfig/cron` scheduler instance
- Default cron: `0 3 * * *` (3 AM daily) — configurable via `CLEANUP_CRON` env var
- Retention window configurable via `CLEANUP_RETENTION_DAYS` env var (default `3`)
- Log deleted counts: `articles_deleted`, `clusters_deleted`
- Same atomic overlap guard as ingest job

### Checklist
- [x] Implement `internal/jobs/cleanup.go` with retention window config
- [x] Add `CLEANUP_CRON` env var (default: `0 3 * * *`)
- [x] Add `CLEANUP_RETENTION_DAYS` env var (default: `3`)
- [x] Wire into fx lifecycle alongside ingest scheduler
- [x] Log deleted row counts per run
- [x] Add `last_cleanup_at` to `/health` response

---

## 9. Analytics & Visitor Tracking

### Goal
Understand real traffic patterns without introducing GDPR complexity or third-party scripts that slow the frontend. Privacy-first: no cookies, no fingerprinting, no PII.

### Recommended Approach: Plausible Analytics (self-hosted or cloud)

**Option A — Plausible Cloud** (simplest):
- $9/mo for up to 10k pageviews/mo, free up to that on the community plan
- Drop-in `<script>` tag, no cookies, GDPR/CCPA compliant out of the box
- Tracks: pageviews, unique visitors, referrers, countries, device types, custom events

**Option B — Umami (self-hosted)** (free, full control):
- Docker container alongside the app
- Same privacy-first model as Plausible, open source
- Stores data in your own PostgreSQL (can reuse existing DB or separate instance)
- Slightly more ops overhead

**Option C — Cloudflare Analytics** (free if behind Cloudflare):
- Zero config, no script tag needed
- Limited data (no referrers, no custom events), but enough for basic traffic monitoring

### What to Track
| Event | Why |
|-------|-----|
| Pageview | Daily/weekly unique visitors |
| Topic click | Which topics drive engagement |
| Article click (outbound) | Which clusters get reads |
| Search query (if implemented) | Content gap discovery |

### Frontend Integration (Plausible / Umami)
```html
<!-- In index.html <head> -->
<script defer data-domain="yourdomain.com" src="https://plausible.io/js/script.js"></script>
```

Custom events via JS:
```ts
// Track outbound article click
window.plausible?.('Article Click', { props: { topic: cluster.category, source: article.source } })
```

### Checklist
- [ ] Decide: Plausible Cloud vs Umami self-hosted vs Cloudflare Analytics
- [ ] Add analytics script tag to `frontend/index.html`
- [ ] Track custom events: topic filter change, article outbound click
- [ ] Add `VITE_ANALYTICS_DOMAIN` env var (empty = analytics disabled in local dev)
- [ ] Verify no PII is sent (no user IDs, no email, no IP stored)
- [ ] If self-hosting Umami: add to `docker-compose.yml`, provision DB schema

---

## 10. Suggested README Additions

The current backend `README.md` is MVP-scoped and needs expansion. Suggested new sections:

1. **Prerequisites** — Go version, Node version, PostgreSQL + pgvector, Redis, ONNX runtime lib, model files download instructions
2. **Quick Start** — step-by-step from zero to running locally
3. **Environment Variables** — full table with name, required/optional, description, example value
4. **ML Models** — what model is used, how to download `libonnxruntime.so` and `model.onnx`, link to HuggingFace model card
5. **Database Setup** — `CREATE EXTENSION vector`, migration commands
6. **API Reference** — all endpoints, parameters, response shapes (or link to OpenAPI spec)
7. **Architecture Diagram** — simple ASCII diagram of the ingestion → clustering → serve pipeline
8. **Deployment** — link to this document or inline summary
9. **Contributing** — code style, test requirements, PR process

---

## 11. Stale Embedding Recovery on Startup

### Problem
The Dispatcher is a pure in-memory job queue. On server restart, any queued embedding jobs are lost — articles remain `pending` in the DB forever with no code to recover them. Articles mid-flight (`processing`) when the server crashes are also orphaned.

### Root Cause
- `embedding_status = 'pending'` is the DB default on insert
- `pending → processing → ready/failed` transitions happen inside the in-memory worker
- No startup reconciliation exists

### Fix (3 small changes)
- `postgres/embeddings.go`: add `MarkProcessingAsFailed` (bulk UPDATE) and `GetPendingArticleIDs` (SELECT)
- `pipeline.go`: add `RecoverStaleEmbeddings()` — marks `processing → failed`, re-enqueues all `pending`
- `modules.go`: call `pipeline.RecoverStaleEmbeddings()` in the dispatcher `OnStart` hook after `dispatcher.Start()`

> Graceful shutdown already works correctly — `wg.Wait()` lets in-flight jobs finish before the process exits, so `processing` items always resolve cleanly on SIGTERM. Only crashes leave orphaned `processing` rows.

### Checklist
- [x] Add `MarkStaleEmbeddingsAsFailed` to `internal/storage/postgres/embeddings.go` (resets both `processing` and `pending` → `failed`)
- [x] Add `RecoverStaleEmbeddings()` to `internal/jobs/pipeline.go`
- [x] Wire call into dispatcher `OnStart` in `internal/app/modules.go`

---

## Progress Tracker

| Area | Status | Priority |
|------|--------|---------|
| Docker / containerization | ✅ Done (multi-stage backend, nginx frontend, reverse proxy, compose) | High |
| Scheduled ingestion | ✅ Done (robfig/cron, `INGEST_CRON` env var) | High |
| Feed caching (Redis) | ✅ Done (`feed:home:p{n}`, `feed:topic:{X}:p{n}`, post-ingest invalidation) | High |
| Stats endpoint | ✅ Removed | Medium |
| Rate limiting middleware | ✅ Done (Redis-backed, per-IP, 30/60 req/min) | High |
| Atomic feed updates | ✅ Done (unnest batch rescore, feed_version in Redis + response, frontend ref-check) | Medium |
| CORS restriction | ✅ Done (`ALLOWED_ORIGINS` env var, 10s timeout) | Medium |
| Prometheus metrics | ❌ Not started | Medium |
| Extended /health endpoint | ✅ Done (redis, article/cluster counts, embedding_enabled) | Medium |
| CI/CD pipeline | ❌ Not started | Medium |
| DB cleanup job (3-day retention) | ✅ Done (`CLEANUP_CRON`, `CLEANUP_RETENTION_DAYS`, COALESCE fallback) | High |
| Analytics / visitor tracking | ❌ Not started | Medium |
| Stale embedding recovery on startup | ✅ Done (`processing`+`pending` → `failed` on startup, logs counts) | Low |
| .env.example files | ✅ Done | Low |
| README expansion | ❌ Not started | Low |
| OpenAPI / Swagger spec | ❌ Not started | Low |
