# Security Policy

Jadro runs in production at [jadro.mk](https://jadro.mk), so security reports matter to us.

## Reporting a vulnerability

**Please do not open a public issue, discussion or pull request for security problems.**

Report privately through GitHub: **Security → Report a vulnerability** on this repository ([direct link](https://github.com/MatejGadjo/jadro/security/advisories/new)).

Please include:

- what is affected (endpoint, file, configuration)
- steps to reproduce or a proof of concept
- the impact you think it has

You can expect an acknowledgement within **7 days** and a fix or mitigation plan within **30 days** for confirmed issues. We'll credit you in the advisory unless you'd rather stay anonymous.

## Scope

In scope: the code in this repository and the default configuration it ships (`docker-compose.yml`, `nginx/nginx.conf`).

Out of scope:

- denial-of-service through sheer traffic volume (volumetric DDoS)
- findings that require a compromised host, or a misconfiguration contradicting [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- automated scanner output without a demonstrated impact
- testing against jadro.mk that degrades the service for other people. Please test against your own instance (`make up`).

## Supported versions

Only the latest release on `master` receives security fixes.
