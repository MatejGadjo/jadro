# Contributing to Jadro

Thanks for helping! Bug reports, new news sources, fixes, tests and docs are all welcome.

## Before you start

- For anything bigger than a small fix, **open an issue first** so we can agree on the approach.
- Security problems go through [SECURITY.md](SECURITY.md), not issues.
- Be kind: see the [Code of Conduct](CODE_OF_CONDUCT.md).

## Setting up

Follow the Quick start in the [README](README.md) (`make setup && make models && make up`). For hot reload, use `make dev-deps`, `make dev-backend` and `make dev-frontend`.

Tooling:

- Go 1.25+, and [golangci-lint v2](https://golangci-lint.run/welcome/install/) for `make lint`
- Node 20+ (CI uses 24)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), only if you change SQL queries

## Making changes

1. Fork the repo and branch off `master` (`fix/broken-image-probe`, `feat/source-yaml`).
2. Keep pull requests focused: one change per PR.
3. Add or update tests for behaviour you change.
4. Run `make lint` and `make test` before pushing. CI runs the same checks.
5. Open a PR and fill in the template.

### Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/): `feat: …`, `fix: …`, `docs: …`, `refactor: …`, `test: …`, `chore: …`. Explain *why* in the body when it isn't obvious.

## Backend notes

- **Layering:** `rest/handlers` → `service` → `storage` (sqlc). Handlers stay thin; services hold logic; packages depend on small interfaces declared where they are used (`dependency.go` files).
- **Dependency injection:** everything is wired in `internal/app/app.go` with uber/fx.
- **SQL:** never edit `internal/storage/sqlc/*.go` by hand, because it is generated.
  1. Edit `internal/storage/sqlc/queries/*.sql`
  2. Run `sqlc generate` in `backend/`
  3. Expose the new query through a wrapper in `internal/storage/*.go`, mapping to `domain` types
- **Migrations:** add a new numbered file in `internal/storage/migrations/` (goose format, with `-- +goose Up`/`Down`). Never edit a migration that has already shipped. Migrations run automatically at startup.
- **Topics:** a new `CategoryMethod` or topic must also be allowed by the DB check constraints. `TestCategoryMethodsAllowedByDB` guards the methods.

## Frontend notes

- All API calls live in `src/api/` and are validated with zod. Components use hooks from `src/hooks/`, never axios directly.
- Styling is Tailwind only. shadcn/ui primitives live in `src/components/ui/`.
- Keep the product constraints in [frontend/AGENTS.md](frontend/AGENTS.md) in mind (cluster-based feed, no accounts, no tracking by default).

## Adding a news source

Open an issue with the **New news source** template, or send a PR adding a migration that inserts into `sources` (see `00007_seed_sources.sql` for the columns). Please only add outlets that publish a public RSS feed.

## AI coding assistants

[AGENTS.md](AGENTS.md) (with `CLAUDE.md` pointing to it) describes the codebase for AI tools. You're welcome to use them, but you are responsible for every line you submit. Please review and test it like your own code.
