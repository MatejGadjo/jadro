# Trusted proxy ranges

`nginx/nginx.conf` includes every `*.conf` file in this directory inside the
`http` block. Use it to trust the IP ranges of a CDN or load balancer in front of
your server, so rate limiting sees the real visitor IP and not the CDN's.

- **No CDN:** leave this directory as it is. Only the Docker network is trusted.
- **Cloudflare:** `cp cloudflare.conf.example cloudflare.conf`
- **Another CDN:** create `mycdn.conf` with one `set_real_ip_from <cidr>;` line
  per published range.

`*.conf` files here are gitignored. They are deployment config, not project code.
After changing them, recreate the container: `docker compose up -d --force-recreate nginx`.
