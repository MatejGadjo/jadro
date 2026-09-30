# Deploying Jadro

This guide runs Jadro on a single small VPS (2 vCPU / 4 GB RAM is plenty) behind Cloudflare. That's the setup jadro.mk uses. Adapt it to your provider.

```
visitor ─▶ Cloudflare (DNS proxy, WAF, cache) ─▶ VPS :443 host nginx (TLS) ─▶ 127.0.0.1:8000 docker nginx ─▶ backend / frontend
```

## 1. Server

1. Ubuntu LTS with Docker Engine + Compose v2 installed.
2. Create a non-root user for deploys, and log in with SSH keys only (`PasswordAuthentication no`, `PermitRootLogin no`).
3. **Firewall:** allow SSH only from your own IP (or use Tailscale/WireGuard), and allow 80/443 **only from [Cloudflare's IP ranges](https://www.cloudflare.com/ips/)**. Otherwise anyone who finds the server IP can bypass Cloudflare's protection. Use your provider's cloud firewall or `ufw`. Cloudflare Tunnel (no open ports at all) is an even stronger option.
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

For analytics, put `VITE_GA_MEASUREMENT_ID=G-…` in `frontend/.env` before building. Vite bakes it into the bundle.

## 3. TLS proxy on the host

Terminate TLS on the host (nginx + certbot, or a Cloudflare Origin Certificate) and proxy to `127.0.0.1:8000`. **Append** to `X-Forwarded-For` so the container can recover the real visitor IP:

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

The container's `nginx/nginx.conf` trusts the Docker network and Cloudflare's ranges and walks `X-Forwarded-For` from the right. If you use a different CDN, add its ranges as `set_real_ip_from` lines there, otherwise every visitor behind the same CDN node shares one rate-limit bucket.

## 4. Cloudflare

- DNS records proxied (orange cloud), SSL mode **Full (strict)**.
- **Security → WAF → Rate limiting rule** on `/api/*` (for example 100 requests / 10 s per IP → block) as an outer layer in front of nginx.
- Bot Fight Mode on. Keep **Under Attack Mode** in mind as the emergency switch.
- Cache rules: static assets (`/assets/*`) cache everything. The API is not cached at the edge (Redis handles that).
- Refresh the `set_real_ip_from` list in `nginx/nginx.conf` occasionally from https://www.cloudflare.com/ips/.

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

- [ ] Origin firewall allows 80/443 only from your CDN, and SSH only from you
- [ ] `APP_ENV=production`, strong `POSTGRES_PASSWORD`, `ALLOWED_ORIGINS` set
- [ ] Postgres/Redis/nginx ports still bound to `127.0.0.1`
- [ ] Host proxy appends `X-Forwarded-For`; `nginx/nginx.conf` trusts your CDN ranges
- [ ] CDN rate-limiting rule on `/api/*`
- [ ] Branch protection + protected `production` environment if you use the deploy workflow
- [ ] Backups run and have been restored at least once
