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

Package-by-feature. Dependencies point one way:
`cmd/api` (wiring) → `httpapi` (Gin handlers, DTOs) → feature packages → `db` / `storage` / `realtime`.

- Each feature package (`auth`, `user`, `tweet`, `feed`, `search`, `discovery`, `notification`, `message`, `assistant`, `media`) exposes a `Service` built with `New(...)` and returns concrete types.
- Interfaces are declared by their consumer: `httpapi/services.go` for handlers; small private interfaces inside features for storage/notifier dependencies.
- `domain` holds the read models shared across features (`domain.User`, `domain.Tweet`, …) so features don't import each other in cycles.
- HTTP DTOs (`httpapi/responses.go`) are separate from domain types; e.g. email is only exposed to its owner.

## Media Upload Flow

Media (images/videos) are uploaded directly to S3 via presigned URLs:
1. Client calls `POST /api/v1/uploads/presign` with `{contentType, folder, contentLength}`
2. API returns `{presignedUrl, objectKey}`; the key is `{folder}/{userId}/{uuid}{ext}`
3. Client PUTs the file directly to S3 using the presigned URL (same `Content-Type`/size; both are signed)
4. Client includes the `objectKey` when creating a tweet or updating profile

Media is served through CloudFront CDN.

## API Behavior (Important)

- Pagination uses cursor model:
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

Run all tests:

```bash
go test ./...
```

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

Integration test note:
- `internal/db/store_integration_test.go` uses `TEST_DATABASE_URL`
- It is skipped when `TEST_DATABASE_URL` is not set

## Main Directories

- `cmd/api`: entrypoint (wiring only)
- `internal/db/migrations`: schema migrations (embedded and applied on startup)
- `internal/db/queries`: sqlc query source; `internal/db`: generated code + `Store` (transactions); `internal/db/dbtest`: store fake for unit tests
- `internal/httpapi`: router, handlers, request/response DTOs
- `internal/<feature>`: business logic per feature (see Architecture)
- `internal/domain`: shared read models
- `internal/realtime`: per-user event fan-out (Redis pub/sub across instances)
- `internal/storage`: S3 presign/delete + public URLs; `internal/embedding`: embedding job publisher
- `internal/middleware`, `internal/apperr`, `internal/config`, `internal/token`, `internal/logger`
