# Deploying Jadro

Jadro is a plain Docker Compose stack, so it runs on any machine with Docker: a VPS from any provider, a home server, or a cloud VM. This guide describes a single small server (2 vCPU / 4 GB RAM is plenty) with a TLS proxy on the host. A CDN in front is optional but recommended for public instances.

```
visitor ─▶ [optional CDN / WAF] ─▶ server :443 host proxy (TLS) ─▶ 127.0.0.1:8000 docker nginx ─▶ backend / frontend
```

## 1. Server

1. Ubuntu LTS with Docker Engine + Compose v2 installed.
2. Create a non-root user for deploys, and log in with SSH keys only (`PasswordAuthentication no`, `PermitRootLogin no`).
3. **Firewall:** allow SSH only from your own IP (or use Tailscale/WireGuard). Open 80/443 to everyone if you have no CDN. If you use a CDN, open them **only to the CDN's published IP ranges**, otherwise anyone who finds the server IP can bypass it. Use your provider's cloud firewall or `ufw`.
4. Enable unattended security upgrades.

## 2. App

```bash
git clone https://github.com/MatejGadjo/jadro.git /srv/jadro && cd /srv/jadro
cp .env.example .env         # then edit, see below
./scripts/download-models.sh
docker compose up -d --build
```

Production `.env` (in addition to the defaults):

```bash
APP_ENV=production
POSTGRES_PASSWORD=<long random string>          # openssl rand -base64 32
ALLOWED_ORIGINS=https://example.com,https://www.example.com
INGEST_ON_START=false
# optional Web Push
VAPID_PUBLIC_KEY=...  VAPID_PRIVATE_KEY=...  VAPID_SUBJECT=mailto:you@example.com
```

The compose stack binds nginx, Postgres and Redis to `127.0.0.1` only. **Never** change those to `0.0.0.0`. Reach the database through an SSH tunnel when you need it.

**Analytics (optional, off by default).** For Google Analytics 4, put `VITE_GA_MEASUREMENT_ID=G-…` in `frontend/.env` before building (Vite bakes it into the bundle). Then allow its origins in the Content-Security-Policy through `CSP_SCRIPT_SRC` / `CSP_CONNECT_SRC` in `.env` (see `.env.example`). Any other provider (Plausible, Umami, a CDN's beacon) works the same way: add its origins to those two variables.

## 3. TLS proxy on the host

Terminate TLS on the host (nginx + certbot, Caddy, Traefik, or your CDN's origin certificate) and proxy to `127.0.0.1:8000`. **Append** to `X-Forwarded-For` so the container can recover the real visitor IP:

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
    # ssl_certificate ... ;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

The container's `nginx/nginx.conf` trusts only the Docker network by default and walks `X-Forwarded-For` from the right to find the real visitor.

## 4. A CDN in front (optional)

Any CDN or reverse-proxy service works: Cloudflare, Bunny, Fastly, CloudFront, and so on. Whichever you use:

1. **Trust its IP ranges** so rate limits apply per visitor, not per CDN node. Drop a `*.conf` file into `nginx/real-ip.d/` (see [its README](../nginx/real-ip.d/README.md)). A ready-made Cloudflare file ships as `cloudflare.conf.example`.
2. **Lock the origin** firewall to the CDN's ranges (step 1.3).
3. **Add an edge rate limit** on `/api/*` (for example 100 requests / 10 s per IP) as an outer layer in front of nginx, and turn on the provider's bot protection.
4. **Cache** static assets (`/assets/*`) at the edge. Don't cache `/api/*` there, because Redis handles that.

Without a CDN, nginx's own per-IP limits still apply. You just have less protection against large-scale traffic.

## 5. Continuous deployment (optional)

`.github/workflows/deploy.yml` runs tests, then SSHes into the server and runs `git fetch && git reset --hard origin/master && docker compose up -d --build`. It only runs for the upstream repository. In a fork, change the `if:` condition and set these secrets:

| Secret | Value |
|---|---|
| `VPS_HOST` | server IP or hostname |
| `VPS_USER` | the deploy user |
| `VPS_SSH_KEY` | a private key used only for deploys |
| `VPS_APP_DIR` | e.g. `/srv/jadro` |

Protect it with a GitHub **Environment** named `production` that has required reviewers, and branch protection on `master`.

## 6. Operations

- **Backups:** `docker compose exec -T postgres pg_dump -U news newsaggregator | gzip > backup.sql.gz` on a cron, copied off the server. Articles are short-lived (retention days), so backups mostly matter for push subscriptions and sources.
- **Monitoring:** an uptime check on `https://example.com/api/health` (returns `ok`, DB/Redis status and counts).
- **Logs:** `docker compose logs -f backend` (JSON in production).
- **Updates:** pull and rebuild. Migrations run automatically on startup.

## Security checklist

- [ ] SSH only from you; with a CDN, origin 80/443 only from the CDN
- [ ] `APP_ENV=production`, strong `POSTGRES_PASSWORD`, `ALLOWED_ORIGINS` set
- [ ] Postgres/Redis/nginx ports still bound to `127.0.0.1`
- [ ] Host proxy appends `X-Forwarded-For`; with a CDN, its ranges are in `nginx/real-ip.d/`
- [ ] With a CDN, an edge rate-limiting rule on `/api/*`
- [ ] Branch protection + protected `production` environment if you use the deploy workflow
- [ ] Backups run and have been restored at least once
