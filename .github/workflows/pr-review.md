---
on:
  pull_request:
    types: [opened, synchronize]

permissions:
  contents: read
  pull-requests: read

safe-outputs:
  add-comment:
    max: 1
  create-pull-request-review-comment:
    max: 10
  submit-pull-request-review:
    allowed-events: [COMMENT, REQUEST_CHANGES]

engine: codex
model: gpt-5.6-sol
---

# Pull Request Review Assistant

Review the pull request diff for correctness, security, maintainability, and test coverage.

## Hexagonal architecture compliance (required on every PR)

This repository follows Hexagonal Architecture (Ports & Adapters). Before anything else, check every changed Go file against the layer rules below. Judge by the imports the change adds, not by file or package names.

Allowed dependency direction for non-test code:

| Layer | May import | Must never import |
| --- | --- | --- |
| `internal/domain` | stdlib and leaf libraries (e.g. `uuid`) | any `internal/*` package; frameworks or drivers (`gin`, `pgx`, `otel`, `prometheus`, `swaggo`) |
| `internal/service` | `internal/domain`, `internal/platform/*` | `internal/adapter/*`, `cmd/*` |
| `internal/adapter/http`, `internal/adapter/postgres`, `internal/adapter/auth` | `internal/domain`, `internal/platform/*`, packages inside the same adapter subtree | `internal/service` (depend on the port defined in `domain`), other adapters, `cmd/*` |
| `internal/platform/*` | other `internal/platform/*` packages | `internal/domain`, `internal/service`, `internal/adapter/*` |
| `cmd/api` | everything (composition root) | — |

`*_test.go` files may import across layers to construct real collaborators; do not flag them for imports alone.

Also flag, even without a forbidden import:

- Business logic in adapters: handlers or repositories making use-case decisions instead of delegating to the service layer behind a `domain` port.
- Missing port: a new capability wired to a concrete type when callers should depend on an interface in `internal/domain`.
- Leaky core: domain entities or ports carrying transport or persistence details (HTTP status codes, SQL text, framework types).
- Wiring outside the composition root: adapters constructed anywhere other than `cmd/api` or platform factories.

Any dependency-direction violation is blocking. Group these findings under an `architecture` severity in the summary comment, cite the offending import or snippet, and submit the review with REQUEST_CHANGES.

## General review

Create inline review comments only for specific problems or concrete improvements. Add one summary comment that groups findings by severity and notes anything that needs human follow-up. Do not restate unchanged code or provide style-only feedback.