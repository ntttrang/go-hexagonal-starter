# Go Hexagonal Starter
![Go Hexagonal Starter - CI](https://github.com/ntttrang/go-hexagonal-starter/actions/workflows/ci.yml/badge.svg?branch=develop)

Production-ready Go microservice using **Hexagonal Architecture** (Ports & Adapters), Gin, PostgreSQL, JWT auth, Swagger, structured logging, Prometheus metrics, and OpenTelemetry tracing.

## Features

- User registration, login (JWT), and protected CRUD APIs
- Liveness (`/healthz`), readiness (`/readyz`), Prometheus (`/metrics`), Swagger (`/swagger/index.html`)
- OpenTelemetry traces (OTLP) with request-ID and `trace_id`/`span_id` log correlation
- SQL migrations on startup (golang-migrate)
- Docker Compose app stack; observability stack (Grafana, Prometheus, Loki, Tempo, OTEL Collector) lives in [IaC](https://github.com/ntttrang/IaC)
- GitHub Actions CI + deploy to Docker Hub, AWS ECR, and ECS Fargate (ADOT sidecar)

## Architecture

```
cmd/api                 composition root
internal/domain         entities + ports (no framework deps)
internal/service        application use cases
internal/adapter/http   Gin handlers + middleware
internal/adapter/postgres
internal/adapter/auth   JWT issuer
internal/platform       config, db, logger, metrics, tracing
migrations              SQL migrations
deployments/ecs         ECS Fargate task definition
```

## Quick start (Docker Compose)

```bash
cp .env.example .env
docker compose up --build                 # API + Postgres
```

| Service | URL |
|---------|-----|
| API | http://localhost:8085 |
| Swagger | http://localhost:8085/swagger/index.html |

The observability stack (Grafana, Prometheus, Loki, Tempo, OTEL Collector) runs from the **[IaC repo](https://github.com/ntttrang/IaC)** — see [Observability](#observability).

### Example flow

```bash
# Register
curl -s -X POST http://localhost:8085/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","name":"Alice","password":"password123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8085/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}' | jq -r .token)

# List users
curl -s http://localhost:8085/api/v1/users -H "Authorization: Bearer $TOKEN"
```

## Observability

Three pillars are wired end-to-end and correlated via `trace_id`:

| Pillar | Local | Production |
|--------|-------|------------|
| **Traces** | App → OTEL Collector → Tempo | App → ADOT sidecar → Tempo / Grafana Cloud (OTLP) |
| **Metrics** | Prometheus scrapes `/metrics` | Same scrape target (or AMP); CloudWatch for container metrics |
| **Logs** | stdout → Promtail → Loki | stdout → CloudWatch Logs (`awslogs`), JSON includes `trace_id` |

### App instrumentation

- **Tracing**: OpenTelemetry SDK + `otelgin` middleware; disabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is empty
- **Metrics**: HTTP RED, Go runtime/process, DB pool gauges, `auth_login_total`, `user_operations_total`
- **Logs**: JSON `slog` with `service`, `env`, `request_id`, and span-derived `trace_id` / `span_id`

### Running the observability stack

The Compose overlay (`docker-compose.observability.yml`) and all collector, datasource, and dashboard configs now live in the **[IaC repo](https://github.com/ntttrang/IaC)**. Start the stack from there:

```bash
git clone https://github.com/ntttrang/IaC && cd IaC
# follow its README to bring up Grafana, Prometheus, Loki, Tempo, and the OTEL Collector
```

Then point the app at the collector via `OTEL_EXPORTER_OTLP_ENDPOINT` (the overlay uses `otel-collector:4317`). The app itself (API + Postgres) still starts from this repo with `make up`; leave the endpoint unset to keep tracing off.

### Grafana

1. With the stack running from [IaC](https://github.com/ntttrang/IaC), open http://localhost:3000
2. Datasources (Prometheus, Loki, Tempo) are auto-provisioned under *Observability*
3. Dashboard **Go Hexagonal Starter — RED + Logs** shows request rate, latency, errors, auth, DB pool, and logs
4. From a Loki log line, use the **View Trace** link to jump to Tempo; from a Tempo span, use *Logs for this span*

### Production (ECS)

[`deployments/ecs/task-definition.json`](deployments/ecs/task-definition.json) runs an **ADOT** sidecar. The API sends OTLP to `localhost:4317`. Set:

- `OTEL_BACKEND_ENDPOINT` — your Tempo / Grafana Cloud OTLP gRPC endpoint
- Secrets Manager secret `go-hexagonal/otel-backend-token` for auth headers
- Optionally switch the collector exporter to AWS X-Ray / AMP and attach the corresponding IAM permissions on `ecsTaskRole`

Leave `OTEL_EXPORTER_OTLP_ENDPOINT` unset to keep tracing off.

### Generating traffic to verify dashboards

[`scripts/loadtest.js`](scripts/loadtest.js) drives the API with [k6](https://k6.io) to populate all three pillars (metrics, logs, traces) so you can confirm the dashboards are wired end-to-end. With the IaC observability stack running and `make up` for the app:

```bash
brew install k6                      # macOS; see k6 docs for other platforms
make loadtest                        # ~2 min, ramps to 20 VUs
K6_VUS=50 RUN_ID=2 BASE_URL=http://localhost:8085 make loadtest   # overrides
```

Then check the RED + Logs dashboard in Grafana, or run the Prometheus / Loki / Tempo queries above.

## Local development

Requirements: Go 1.25+, Docker, (optional) golangci-lint, swag, migrate CLI.

```bash
cp .env.example .env
# start only Postgres
docker compose up -d postgres

export $(grep -v '^#' .env | xargs)
make run
```

Useful targets:

| Target | Description |
|--------|-------------|
| `make test` | Unit tests |
| `make test-integration` | Integration tests (needs Postgres via `TEST_DATABASE_URL`) |
| `make lint` | golangci-lint |
| `make swag` | Regenerate OpenAPI docs |
| `make build` | Build binary to `bin/api` |
| `make up` / `make down` | Start / stop app + Postgres |

The observability stack runs from the **[IaC repo](https://github.com/ntttrang/IaC)** — see [Observability](#observability).

## Configuration

See [`.env.example`](.env.example). Important variables:

- `JWT_SECRET` — required, min 16 characters
- `DB_*` — PostgreSQL connection
- `MIGRATIONS_PATH` — default `file://migrations` (Compose uses `file:///app/migrations`)
- `OTEL_EXPORTER_OTLP_ENDPOINT` — OTLP gRPC endpoint (empty = tracing off)
- `OTEL_SERVICE_NAME` / `SERVICE_VERSION` — resource attributes
- `OTEL_TRACES_SAMPLER_ARG` — sample ratio `0.0`–`1.0`

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | No | Create user |
| POST | `/api/v1/auth/login` | No | Obtain JWT |
| GET | `/api/v1/users` | Bearer | List users |
| GET | `/api/v1/users/:id` | Bearer | Get user |
| PUT | `/api/v1/users/:id` | Bearer | Update user |
| DELETE | `/api/v1/users/:id` | Bearer | Delete user |
| GET | `/healthz` | No | Liveness |
| GET | `/readyz` | No | Readiness (DB ping) |
| GET | `/metrics` | No | Prometheus metrics |
| GET | `/debug/pprof/*` | No | Go pprof (non-production only) |

## CI/CD

### CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml))

On PR/push: `go vet`, golangci-lint, unit + integration tests, build.

### Deploy ([`.github/workflows/deploy.yml`](.github/workflows/deploy.yml))

On push to `main` / tags `v*`:

1. Build multi-stage image
2. Push to **Docker Hub** and **Amazon ECR**
3. Render [`deployments/ecs/task-definition.json`](deployments/ecs/task-definition.json)
4. Deploy to **ECS Fargate**

### Required GitHub secrets / variables

**Secrets**

| Name | Purpose |
|------|---------|
| `DOCKERHUB_USERNAME` | Docker Hub login |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `AWS_ROLE_ARN` | IAM role for OIDC assume-role |

**Variables** (optional overrides)

| Name | Default |
|------|---------|
| `AWS_REGION` | `us-east-1` |
| `ECR_REPOSITORY` | `go-hexagonal-starter` |
| `DOCKERHUB_IMAGE` | `your-dockerhub-user/go-hexagonal-starter` |
| `ECS_CLUSTER` | `go-hexagonal-cluster` |
| `ECS_SERVICE` | `go-hexagonal-service` |

Update placeholder ARNs and RDS endpoint in the ECS task definition before first deploy. Store `DB_PASSWORD`, `JWT_SECRET`, and `OTEL_BACKEND_TOKEN` in AWS Secrets Manager and point the task definition `secrets` entries at those ARNs.

## License

MIT
