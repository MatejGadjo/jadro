# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security
- Go toolchain 1.26.9 and golang.org/x/net v0.60.0 (fixes GO-2026-6599..6611
  in net/http, html/template, crypto/tls and mime/multipart).

### Changed
- Go 1.26 (toolchain go1.26.8): fixes 25 standard-library vulnerabilities
  reported by govulncheck for 1.26.0; dependencies updated (minor/patch).
- `onnxruntime_go` stays at v1.25.x to match ONNX Runtime 1.23.2.

## [0.1.0] - First public release

### Added
- One-command local setup: `make setup && make models && make up`, with
  checksum-verified model downloads, `.env.example` files and `INGEST_ON_START`.
- Configurable host ports, Docker subnet, trusted proxies and ONNX Runtime
  architecture (x64 / aarch64).
- nginx per-IP rate and connection limits, real client IP resolution behind
  the host proxy and an optional CDN (`nginx/real-ip.d/`).
- Frontend unit tests (Vitest), golangci-lint configuration, CI for lint,
  tests, vulnerability and secret scanning.
- Open-source project files: license, contributing guide, security policy,
  code of conduct, architecture and deployment docs.

### Changed
- Go module path is now `github.com/MatejGadjo/jadro`.
- `PATCH /api/articles/:id/clear-image` only clears an image after the server
  verifies it is really broken.
- `/api/feed` rejects `page` values outside 0–100.
- Requests are limited to 16 KiB bodies; server read/idle timeouts added.

### Fixed
- Articles from forced-category sources (e.g. dedicated sports and business
  outlets) were rejected by a stale database check constraint.
