# Production Readiness Roadmap

---

## What's Built (Architecture Reference)

**Stack**: Cloudflare → nginx (TLS, SSL terminator) → docker-compose nginx (:8000) → Go/Gin backend (:8080) → PostgreSQL + pgvector + Redis

**Deployed at**: reservelt.co (temp) → jadro.mk (when purchased). Hetzner CX23, Helsinki. Ubuntu 24.04 LTS. Repo cloned at `/opt/jadro`.

**Scheduler**: robfig/cron in-process. `INGEST_CRON` (default `*/15 * * * *`), fires on startup, atomic overlap guard, rescore after every ingest. Cleanup job: `CLEANUP_CRON` (default `0 3 * * *`), `CLEANUP_RETENTION_DAYS=3` (deletes cluster_articles → articles → empty clusters).

**Caching**: Redis read-through on `GET /api/feed`. Keys: `feed:home:p{n}`, `feed:topic:{X}:p{n}`, 5 min TTL. Post-ingest `DeleteByPattern("feed:*")`. `REDIS_URL` empty = caching disabled.

**Feed versioning**: `feed_version` (unix timestamp) in every `/api/feed` response, written to `meta:feed_version` in Redis after each rescore. Frontend `select` ref-check skips re-render when version unchanged (`placeholderData: keepPreviousData`).

**Rate limiting**: GCRA per-IP, Redis-backed, fails open. Feed: 30/min. Clusters/articles: 60/min. Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`.

**Security**: `ALLOWED_ORIGINS` env var (empty = wildcard `*`). 10s context timeout on all handlers.

**Logging**: zap — dev mode = colored console, production (`APP_ENV=production`) = JSON. `LOG_LEVEL` env var overrides level (`debug`/`info`/`warn`/`error`, default `info`).

**Analytics**: GA4 in cookieless mode (`storage: none`, `anonymize_ip`, no ad signals). `VITE_GA_MEASUREMENT_ID` env var — empty = disabled in dev. `trackEvent()` helper in `src/analytics.ts` for custom events. Cloudflare Analytics automatic once behind CF (no code needed).

**ML models**: ONNX runtime + model files volume-mounted on VPS (`ONNX_RUNTIME_LIB_PATH`, `ONNX_MODEL_PATH`, `ONNX_TOKENIZER_PATH`). Graceful degradation if unset → rule-based clustering fallback. libonnxruntime.so ~200 MB, model.onnx ~120 MB — not in git.

**Migrations**: Auto-run on startup via goose + `//go:embed`. pgvector extension handled in migration 00002.

**Stale embeddings**: On startup, `processing` + `pending` rows → `failed`, then re-enqueued.

**CI/CD**: `test.yml` runs Go tests + frontend build on every push/PR. `deploy.yml` gates on both before SSH deploy to VPS (`git pull → docker compose up -d --build`). Dormant until GitHub Actions secrets provisioned.

---

## Open Items

### Deployment
- [x] Redis hosting — self-hosted in docker-compose (already wired, zero config)
- [x] Model file download steps documented in README
- [x] Download ML models on VPS (`/opt/jadro/backend/models/`) — see README for commands
- [ ] Wire up GitHub Actions secrets (`VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `VPS_APP_DIR`) to enable auto-deploy on push
- [ ] Switch domain from reservelt.co → jadro.mk when purchased (update A record + `ALLOWED_ORIGINS`)

### Security
- [x] App behind Cloudflare free plan — Cloudflare Origin Certificate for TLS, Full (strict) mode
- [x] Protect `/internal/*` routes with bearer token — `INTERNAL_API_TOKEN` env var; required in production, no-op when empty in dev. `POST /internal/ingest` triggers a manual ingest cycle.
- [x] Security review completed — constant-time token comparison, startup warning for missing token, 409 on double-ingest

### Monitoring
- [x] `LOG_LEVEL` env var — `debug`/`info`/`warn`/`error`, overrides default per `APP_ENV`.
- [ ] Alerting: set up UptimeRobot free tier → monitor `https://reservelt.co/api/health`

### Analytics
- [x] GA4 wired in cookieless mode — `VITE_GA_MEASUREMENT_ID` env var (empty = disabled in dev). No cookies, no consent banner.
- [x] Cloudflare Analytics — automatic, already active
- [x] Custom events: `topic_select` on nav click, `article_click` on all outbound article links
- [ ] Create GA4 property → get `G-XXXXXXXXXX` → add to `/opt/jadro/.env` → `docker compose up -d --build`

### README
- [x] Prerequisites, Quick Start, env vars table, ML model download instructions, API reference, architecture, deployment guide link

---

## VPS Deployment Guide

### What You Need to Do (account/infra actions)

1. Provision **Hetzner CX22** (€4/mo, 2 vCPU 4 GB RAM) — hetzner.com
2. Point **jadro.mk** to Cloudflare free plan → proxy A record to VPS IP (orange cloud)
3. Create **GA4 property** at analytics.google.com → get `G-XXXXXXXXXX` measurement ID
4. Decide **Redis hosting** (see Step 6 below)

### Step 1 — Harden the VPS

```bash
sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
systemctl restart ssh

curl -fsSL https://get.docker.com | sh
usermod -aG docker $USER
```

### Step 2 — nginx + TLS

```bash
apt install nginx certbot python3-certbot-nginx
certbot --nginx -d jadro.mk -d www.jadro.mk
```

`/etc/nginx/sites-available/jadro`:
```nginx
server {
    listen 80;
    server_name jadro.mk www.jadro.mk;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name jadro.mk www.jadro.mk;

    location / {
        proxy_pass http://localhost:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }

    location /metrics { deny all; }
}
```

### Step 3 — Upload ML Model Files

Model files are not in git — upload once to VPS:

```bash
scp libonnxruntime.so user@VPS_IP:/opt/jadro/models/
scp model.onnx        user@VPS_IP:/opt/jadro/models/
scp tokenizer.json    user@VPS_IP:/opt/jadro/models/
```

| File | Size |
|------|------|
| `libonnxruntime.so` | ~200 MB |
| `model.onnx` | ~120 MB |
| `tokenizer.json` | < 1 MB |

`docker-compose.yml` volume-mounts `/opt/jadro/models` → `/models` inside the container. Path is fixed.

### Step 4 — Production Environment

`/opt/jadro/.env` on the VPS:
```env
APP_ENV=production
LOG_LEVEL=warn
HTTP_ADDR=:8080

DATABASE_URL=postgres://jadro:CHANGE_ME@postgres:5432/jadro?sslmode=disable
REDIS_URL=redis://redis:6379

ALLOWED_ORIGINS=https://jadro.mk,https://www.jadro.mk

ONNX_RUNTIME_LIB_PATH=/models/libonnxruntime.so
ONNX_MODEL_PATH=/models/model.onnx
ONNX_TOKENIZER_PATH=/models/tokenizer.json

INGEST_CRON=*/15 * * * *
CLEANUP_CRON=0 3 * * *
CLEANUP_RETENTION_DAYS=3
```

`/opt/jadro/frontend/.env.production`:
```env
VITE_API_BASE_URL=https://jadro.mk
VITE_GA_MEASUREMENT_ID=G-XXXXXXXXXX
```

### Step 5 — GitHub Actions Secrets

GitHub → Settings → Secrets → Actions:

| Secret | Value |
|--------|-------|
| `VPS_HOST` | VPS IP or domain |
| `VPS_USER` | SSH user (e.g. `ubuntu`) |
| `VPS_SSH_KEY` | Private SSH key for VPS |
| `VPS_APP_DIR` | `/opt/jadro` |

Once set, every push to `main` that passes tests auto-deploys.

### Step 6 — Redis Hosting Decision

| Option | Cost | Pro | Con |
|--------|------|-----|-----|
| Self-hosted (docker-compose) | Free | Already wired, zero config | Data lost on VPS crash |
| Upstash free tier | Free | Survives VPS restart, 10k cmds/day, 256 MB | External dependency |

Self-hosted is the path of least resistance — already in `docker-compose.yml`.

### Step 7 — First Deploy

```bash
git clone https://github.com/MatejGadjo/NewsAggregator /opt/jadro
cd /opt/jadro
docker compose up -d --build
curl http://localhost:8080/api/health
```

### Post-Deploy Checklist

- [ ] Set `VITE_GA_MEASUREMENT_ID` in production env, redeploy
- [ ] Verify GA4 receiving events (GA4 → Realtime view)
- [ ] Set up UptimeRobot free tier → monitor `https://jadro.mk/api/health`
- [ ] Cloudflare SSL/TLS mode → Full (strict)
- [ ] Confirm `ALLOWED_ORIGINS` matches production domain exactly

---

## Progress Tracker

| Area | Status | Priority |
|------|--------|---------|
| Docker / containerization | ✅ Done | High |
| Scheduled ingestion | ✅ Done | High |
| Feed caching (Redis) | ✅ Done | High |
| Rate limiting | ✅ Done | High |
| DB cleanup job (3-day retention) | ✅ Done | High |
| Atomic feed updates + feed_version | ✅ Done | Medium |
| CORS restriction + request timeout | ✅ Done | Medium |
| Extended /health endpoint | ✅ Done | Medium |
| CI/CD pipeline (test gate + SSH deploy) | ✅ Done | Medium |
| Stale embedding recovery on startup | ✅ Done | Low |
| .env.example files | ✅ Done | Low |
| Redis hosting decision | ✅ Self-hosted in docker-compose | High |
| LOG_LEVEL env var | ✅ Done | Medium |
| Cloudflare / nginx in front of app | ✅ Done — Cloudflare + Origin Cert + Full strict | Medium |
| VPS provisioned + site live | ✅ Done — reservelt.co (Hetzner CX23 Helsinki) | High |
| Analytics / visitor tracking | ✅ Done (GA4 cookieless + Cloudflare active) | Medium |
| README expansion | ✅ Done | Low |
| ML models on VPS | ✅ Done | Medium |
| GitHub Actions auto-deploy | ❌ Secrets not set | Medium |
| GA4 property created | ❌ Needs account action | Low |
| UptimeRobot alerting | ❌ Needs account action | Low |
