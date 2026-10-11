# Twitter Go API

Primary backend API for this project.

## Stack

- Go 1.26+, Gin
- PostgreSQL + pgvector, with sqlc (type-safe generated queries) and embedded migrations
- `log/slog` (JSON logs in production), Prometheus metrics
- Redis (optional): shared rate limits and SSE fan-out across several instances
- AWS S3 + CloudFront (presigned media uploads)
- Gemini (assistant and tweet embeddings)
- Swaggo (OpenAPI docs generated from handler annotations)

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

The full contract for clients is [`docs/api-contract.md`](docs/api-contract.md).

- Pagination uses opaque keyset cursors (see `internal/pagination`):
  - Request: `cursor`, `size`
  - Response: `items`, `hasNext`, `nextCursor` (`null` on the last page)
- Timelines (feeds and profile tabs) return `{ kind, tweet, retweet }` feed items; other tweet lists return plain tweets
- Request correlation:
  - every response includes `X-Request-ID`
  - error responses include `requestId` for log correlation
- Health endpoints:
  - `GET /healthz` (liveness)
  - `GET /readyz` (readiness: database, and Redis when configured)

## API Documentation (Swagger)

The API is fully documented using **Swaggo**. You can explore the interactive API reference and test endpoints directly from your browser:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **OpenAPI Spec**: `http://localhost:8080/swagger/doc.json`

Both are served outside production only (`ENVIRONMENT=production` disables them).

The documentation includes request/response schemas, authentication requirements, and error structures. Browsers authenticate with httpOnly session cookies; in Swagger UI, sign in through the web app on localhost (the cookie is shared) or paste an access token into **Authorize** as `Bearer <token>`.

### Generating API Documentation

If you modify handler annotations or response structs, you must regenerate the documentation:

```bash
# Install the swag CLI (same version as go.mod) if not already present
go install github.com/swaggo/swag/cmd/swag@v1.16.6

# Regenerate docs
make swagger  # swag init -g cmd/api/main.go --parseInternal
```

## Local Setup

### Prerequisites

- Go 1.26+
- Postgres with pgvector: `docker compose up -d` from the repo root (Redis comes with it, optional)
- To regenerate code only: [sqlc](https://sqlc.dev) 1.30 and swag (see above)

### Run

```bash
cp app.env.example app.env   # set TOKEN_SYMMETRIC_KEY and GOOGLE_CLIENT_ID
make run                     # applies migrations on startup
```

Default API address: `http://localhost:8080`. Settings can also come from real environment variables, which take precedence over `app.env`.

### Regenerating query code

SQL lives in `internal/db/queries/*.sql`; never edit the generated `*.sql.go`. After changing a query or the schema:

```bash
make sqlc   # regenerates the query code and dbtest.MockStore; CI fails if they're stale
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

## Production Operations

The API runs on EC2 behind nginx, which terminates TLS; port 8080 is never published. See the root README for the AWS setup.
- Liveness: `GET /healthz` (process up). Readiness: `GET /readyz` (database, and Redis when configured); the container health check runs `api healthcheck`, which calls it.
- Health endpoints have no rate limit; `/metrics` is only reachable on the Docker network (nginx returns 404 for it).
- Every response carries `X-Request-ID`; error bodies include `requestId` and every log line for that request carries `request_id`.
