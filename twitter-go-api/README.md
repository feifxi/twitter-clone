# Twitter Go API

Primary backend API for this project.

## Stack

- Go 1.26+
- Gin
- PostgreSQL
- sqlc
- Redis (SSE fanout + distributed rate limiting)
- Zerolog
- AWS S3 (presigned URL media uploads)
- AWS CloudFront (CDN for media)
- Swaggo (Automated OpenAPI 2.0 Docs)

## Architecture

Package-by-feature, with dependencies pointing one way:

```
cmd/api (wiring) ──► internal/api (HTTP) ──► feature packages ──► db / storage / realtime
                                                   │
                                                   └──► domain (shared read models)
```

### Where things live

| Looking for… | Go to |
|---|---|
| **Every endpoint** (method, path, auth, rate limit) | `internal/api/routes.go` |
| A handler (bind input → call service → write DTO) | `internal/api/<feature>_handler.go` |
| Response JSON shapes | `internal/api/responses.go` |
| Interfaces the handlers depend on | `internal/api/services.go` |
| Business logic | `internal/<feature>/service.go`: `auth`, `user`, `tweet`, `feed`, `search`, `discovery`, `notification`, `message`, `assistant`, `media` |
| SQL | `internal/db/queries/*.sql` (generated Go: `internal/db/*.sql.go`) |
| Schema | `internal/db/migrations/` (embedded, applied on startup) |
| Startup and dependency wiring | `cmd/api/main.go` |
| Configuration (env vars) | `internal/config/config.go`, example in `app.env.example` |
| Cross-cutting HTTP middleware | `internal/middleware/` (auth, rate limit, request ID, logging, metrics) |
| Real-time fan-out (SSE) | `internal/realtime/hub.go`, `internal/api/events_handler.go` |
| Background embedding worker | `internal/embedding/worker.go` |
| Cursor pagination | `internal/pagination/` |

Conventions:
- Feature packages expose a `Service` built with `New(...)`, return concrete types and never import Gin.
- Interfaces are declared by their consumer (`api/services.go`, or small private interfaces inside features).
- HTTP DTOs are separate from domain types, so e.g. email is only shown to its owner.

## Media Upload Flow

Media (images/videos) are uploaded directly to S3 via presigned URLs:
1. Client calls `POST /api/v1/uploads/presign` with `{contentType, folder, contentLength}`
2. API returns `{presignedUrl, objectKey}`; the key is `{folder}/{userId}/{uuid}{ext}`
3. Client PUTs the file directly to S3 using the presigned URL (same `Content-Type`/size; both are signed)
4. Client includes the `objectKey` when creating a tweet or updating profile

Media is served through CloudFront CDN.

## API Behavior (Important)

- Pagination uses opaque keyset cursors (see `internal/pagination`):
  - Request: `cursor`, `size`
  - Response: `items`, `hasNext`, `nextCursor`
- Request correlation:
  - every response includes `X-Request-ID`
  - error responses include `requestId` for log correlation
- Health endpoints:
  - `GET /healthz` (liveness)
  - `GET /readyz` (readiness: DB/Redis dependency check)

## API Documentation (Swagger)

The API is fully documented using **Swaggo**. You can explore the interactive API reference and test endpoints directly from your browser:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **OpenAPI Spec**: `http://localhost:8080/swagger/doc.json`

The documentation includes request/response schemas, authentication requirements (Bearer Token), and error structures.

### Generating API Documentation

If you modify handler annotations or response structs, you must regenerate the documentation:

```bash
# Install swag CLI if not already present
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate docs
make swagger  # swag init -g cmd/api/main.go --parseInternal
```

## Local Setup

### Prerequisites

- Go 1.26+
- PostgreSQL
- Redis
- sqlc
- golang-migrate

### Run

```bash
make migrateup
make run
```

Default API address: `http://localhost:8080`

### SQLC Regeneration

If you change queries under `db/query`:

```bash
make sqlc
```

## Testing

```bash
make test                      # unit tests (+ integration tests when TEST_DATABASE_URL is set)
```

Integration tests run the real SQL against Postgres and are skipped without `TEST_DATABASE_URL`. Use any throwaway database, for example:

```bash
docker run -d --name twm-test-pg -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=twitter_test -p 55432:5432 pgvector/pgvector:pg16
TEST_DATABASE_URL='postgres://test:test@localhost:55432/twitter_test?sslmode=disable' make test
```

After editing `internal/db/queries`, run `make sqlc` to regenerate both the query code and `dbtest.MockStore`.

## Production Operations Baseline

For AWS (API Gateway → EC2):
- API Gateway injects `X-Gateway-Secret` header for request authentication
- Liveness probe path: `/healthz`
- Readiness probe path: `/readyz`
- Health endpoints are intentionally outside API rate limiting

Monitoring baseline:
- Track HTTP 5xx rate, p95 latency, and restart count
- Create alert rules for sustained 5xx spikes and failing readiness probes
- Use `requestId` from API error response to correlate app logs for incident triage
