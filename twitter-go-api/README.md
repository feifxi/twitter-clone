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

- `handler -> usecase -> db/service`
- Domain-oriented usecase services with explicit dependencies:
  - auth, user, tweet, feed, search, discovery, notification, message
- Shared error transport model in `internal/apperr`
- DTO/domain mapping separated from generated sqlc models

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
swag init -g cmd/api/main.go
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

- `cmd/api`: entrypoint
- `db/migration`: schema migrations
- `db/query`: sqlc query source
- `internal/db`: generated sqlc code
- `internal/server`: HTTP handlers/routes/response mapping
- `internal/usecase`: business logic services and domain helpers
- `internal/service`: external integrations (S3 storage)
- `internal/config`: environment config
- `internal/token`: JWT logic
