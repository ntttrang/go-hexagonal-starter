# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Ground rules

- Do not commit, push, or open PRs — the user handles all git operations. Conventional commit prefixes (`feat:`, `fix:`, `ci:`) are used in history.
- The module path `github.com/nttttranggo-hexagonal-starter` (triple "t" in "nttttrang") is intentional — use it verbatim in imports.

## Commands

- Unit tests: `make test`. Integration tests sit behind the `integration` build tag and need `TEST_DATABASE_URL` pointing at a live Postgres: `make test-integration`. Plain `go test ./...` silently skips them.
- `make lint` requires golangci-lint **v2.x** — `.golangci.yml` uses the v2 `formatters:` schema and fails to parse with a v1 binary.
- After editing Swagger annotations in handlers, regenerate with `make swag`; the files under `api/docs/` are generated but tracked.
- Local run: `cp .env.example .env` then `make run`, or `make up` for Docker Compose (app + Postgres). Env vars override `.env` values.

## Architecture — hexagonal (ports & adapters)

- `internal/domain` — entities and port interfaces. Only stdlib + `google/uuid` imports allowed; framework types (Gin, pgx, JWT, OTel) never cross this boundary.
- `internal/service` — business logic; depends only on domain ports (+ metrics).
- `internal/adapter/http` — Gin handlers and middleware. Keep handlers thin: bind → call service → `mapError` → respond.
- `internal/adapter/postgres` — the only place with SQL. Map `pgx.ErrNoRows` → `domain.ErrNotFound` and PG error `23505` → `domain.ErrConflict`.
- Error convention: return `domain.Err*` sentinels for expected outcomes, wrap unexpected errors with `%w`; HTTP responses never leak internals (see `mapError` in `internal/adapter/http/errors.go`).

## Gotchas

- Migrations run automatically on startup (golang-migrate). Add new ones as `migrations/NNNNNN_name.up.sql` plus a matching `.down.sql`. Path differs by environment: `file://migrations` locally, `file:///app/migrations` in Docker.
- `JWT_SECRET` shorter than 16 chars fails startup by design.
- The observability stack (Grafana, Prometheus, Loki, Tempo, OTEL collector) lives in the separate IaC repo, not in this one — the README references it but there is no compose file for it here.
