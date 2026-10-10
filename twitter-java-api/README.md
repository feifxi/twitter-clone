# Twitter Java API (optional backend)

A Spring Boot implementation of the same API as [`twitter-go-api`](../twitter-go-api), which is the
**primary backend**. The Java API exists to compare the two stacks: it serves the same contract
(endpoints, JSON, cookies, cursors, error codes) on the same database schema, so the web app runs
against either one unchanged.

## Stack

Java 25 · Spring Boot 4.1 · Spring Security (OAuth2 resource server) · Spring Data JPA + `JdbcClient` ·
Flyway · PostgreSQL + pgvector · Redis (optional) · Spring AI (Gemini) · AWS SDK (S3) · Testcontainers

## Run

```bash
docker compose up -d                       # from the repo root: Postgres + Redis
cp ../twitter-go-api/app.env.example app.env   # the same settings file as the Go API
./mvnw spring-boot:run                     # http://localhost:8080, Swagger at /swagger/index.html
```

It reads the **same environment variables** as the Go API (`DATABASE_URL`, `ACCESS_TOKEN_TTL=15m`, …);
`config/GoStyleEnvironment` translates the few that Spring names differently. Run one backend at a
time on port 8080.

## Test

```bash
./mvnw verify      # unit + integration tests against real Postgres/Redis containers (needs Docker)
```

## How it's organised

Feature packages, each with its controller, service and data access:

| Package | What |
|---|---|
| `auth` | Google sign-in, rotating refresh tokens, session cookies |
| `user` | profiles, onboarding, follows (`User` entity via JPA; follow counters via SQL) |
| `tweet`, `feed` | tweets, likes, retweets; home feeds and profile tabs |
| `search` (+ `discovery`) | user/tweet/hashtag search, trending, suggestions |
| `notification`, `message`, `realtime` | notifications, DMs, the SSE stream (Redis fan-out) |
| `media` | presigned S3 uploads |
| `assistant` | Gemini chat (Spring AI) with RAG over tweet embeddings, the embedding job worker |
| `security`, `shared`, `config`, `ops` | JWT/cookie auth, CSRF origin check, errors, paging, rate limits, health |

Data access is chosen per case: JPA for plain entity CRUD, `JdbcClient` with SQL where JPA gets in
the way (ranked feeds, keyset pagination, counters updated atomically, full-text and vector search).

## Schema

`src/main/resources/db/migration/V1__init_schema.sql` is a copy of the Go API's migration (CI fails
if they differ). On a database the Go API already migrated, Flyway adopts the schema as V1.
