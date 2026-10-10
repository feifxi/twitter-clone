# Modernization Plan

Goal: keep the project a clean, standard, *learnable* monolith. Fix real bugs and security holes first, then raise the architecture and UI to the level of the (soon to be retired) `twitter-microservices` repo, porting only what fits a monolith.

Legend: 🐞 bug · 🔒 security · 🧱 architecture/quality · ✨ feature/UI · 🧹 cleanup

---

## Findings — monolith

### Go API

| # | Kind | Finding | Where |
|---|---|---|---|
| G1 | 🔒 | **Media key IDOR.** `mediaKey`/`avatarKey`/`bannerKey` accept any string, including full URLs. They are stored, and on delete passed to `DeleteFile`, which turns a URL back into an S3 key. A user can embed arbitrary external URLs, or reference someone else's object and delete it by deleting their own tweet. | `usecase/tweet.go`, `usecase/user.go`, `service/storage.go` |
| G2 | 🔒 | Presign size limit is bypassable: `contentLength` is optional and not signed into the PUT. Raw client filename is concatenated into the object key. | `usecase/upload.go`, `service/storage.go` |
| G3 | 🔒 | Every user's **email is exposed** in every public `UserResponse` (search, followers, tweet authors…). | `server/responses.go` |
| G4 | 🔒 | Google login does not check `email_verified`. | `usecase/auth.go` |
| G5 | 🔒 | WebSocket auth passes the JWT in `?access_token=`, and the request logger logs the raw query, so tokens end up in logs. | `message_ws_handler.go`, `logger.go` |
| G6 | 🐞 | RAG query uses `<->` (L2) but the HNSW index is `vector_cosine_ops`, so the **index is never used** and the `< 1.0` threshold has the wrong meaning. | `db/query/embedding.sql` |
| G7 | 🐞 | Assistant SSE writes `data: <chunk>\n\n`; chunks containing `\n` break SSE framing, so **markdown lines are silently dropped** client-side. | `assistant_handler.go` |
| G8 | 🐞 | Login and refresh delete **all** of a user's refresh tokens. Combined with no single-flight refresh on the client, parallel 401s log the user out; logging in on one device logs out all others. | `usecase/auth.go`, `api/axiosInstance.ts` |
| G9 | 🐞 | "Cursor" pagination is a base64 offset. The For-You ranking depends on `NOW()`, so offset pages shift between requests (duplicates or missing tweets). Chronological lists also shift when new rows arrive. | `server/helpers.go`, `db/query/*.sql` |
| G10 | 🐞 | DMs use an in-process WebSocket hub (not fanned out through Redis like SSE), and **API Gateway HTTP APIs don't support WebSocket upgrades**, so real-time DMs don't work in prod. | `message_ws_handler.go`, `infra/terraform/api_gateway.tf` |
| G11 | 🐞 | `000001_init_schema.up.sql` starts with `DROP TABLE … CASCADE` / `DROP EXTENSION`, which is destructive in an *up* migration. | `db/migration` |
| G12 | 🐞 | Like → unlike → like creates duplicate LIKE notifications (no dedupe). | `usecase/tweet.go` |
| G13 | 🧱 | `Server` is a god object: router plus SSE hub, WS hub, Redis listener, and notification hydration. | `server/server.go` |
| G14 | 🧱 | Two error paths: `apperr.Kind` plus `writeError` sniffing `pgx.ErrNoRows` / pg codes leaking up from the DB layer. Validation error shape differs from the microservices one (fine, but consolidate). | `apperr`, `server/errors.go` |
| G15 | 🧱 | Logging uses zerolog (global logger) plus `fmt.Printf` in config. Std-lib `slog` is the modern idiom (as the microservices repo does). | `logger`, `config` |
| G16 | 🧱 | Config: viper with 25 `BindEnv` lines and a package-global SSM flag. Can be a plain env struct. | `config/config.go` |
| G17 | 🧱 | Gemini client is constructed on every assistant request. | `usecase/assistant.go` |
| G18 | 🧱 | Missing endpoints the frontend hacks around: get user **by username** (frontend uses fuzzy search!), a user's likes, replies, and media. | — |
| G19 | 🧱 | Retweets are rows in `tweets` with a `media_type='NONE'` hack; this complicates delete, counts and feeds. A separate `retweets` table plus a `kind: tweet\|retweet` feed item (as in microservices) is cleaner. | schema |
| G20 | 🧹 | `go.mod`: swaggo marked `// indirect` though imported; Gin 1.10 → 1.12; Go 1.24 → 1.26. Dockerfile uses unpinned `alpine:latest`. | — |
| G21 | ✨ | Locally there is no embedder (SQS only), so RAG never works in dev. Lambda uses the deprecated `google-generativeai` SDK. | `service/embedding_publisher.go`, `infra/lambda` |

Already good, so leave as is: sqlc + pgx, `ExecTxAfterCommit` (notify only after commit), batch hydration (no N+1), health/ready split, Redis rate limiter with local fallback, request IDs, Prometheus metrics, Swagger, graceful shutdown, test coverage of usecases and handlers.

### Next.js web

| # | Kind | Finding |
|---|---|---|
| W1 | 🐞 | Unread count: backend returns `{count}`, frontend types it as `number`, so `unreadCount > 0` is always false and **the badge never shows**. |
| W2 | 🐞 | Optimistic follow writes `followedByMe` instead of `isFollowing`. |
| W3 | 🐞 | New tweet is prepended to `['user-feed', id]`, but the profile query key is `['feeds','user',id]`. Query keys are scattered strings. |
| W4 | 🐞 | Profile page resolves `/{username}` by running a **fuzzy user search** and picking a match. |
| W5 | 🐞 | Tweet media is always rendered as `<img>`, even for VIDEO. |
| W6 | 🔒🧱 | Access and refresh tokens in **localStorage** (XSS-exfiltratable). The whole app is client-rendered behind a blocking `AuthInitializer` spinner, so Next.js is used as an SPA and gets no SSR benefit. |
| W7 | 🧱 | Tweet delete has no confirmation. Optimistic-update logic (`useOptimisticTweetUpdate`) is hard to follow, and the debounced toggle lives in component state (lost on unmount). |
| W8 | 🧱 | `QueryClient` is created at module scope. `messages/page.tsx` is 735 lines. `useAuth` and `useCurrentUser` duplicate each other. |
| W9 | 🧹 | Dead or legacy code: `tailwind.config.ts` (ignored by Tailwind v4), `theme/tokens.ts`, `TweetCard.tsx` re-export, `event-source-polyfill`, the `postinstall` type-patch hack, and the brainrot/premium/TikTok pages that hotlink third-party images. |

### Repo / infra / CI

| # | Kind | Finding |
|---|---|---|
| R1 | 🧹 | `twitter-java-api` lags the Go API in features (no DMs, assistant, presign). It has committed `.idea/` and a manual-only Azure workflow, plus `.github/modernize/java-upgrade` leftovers. **Decision: keep it, and bring it to parity with the Go API in the final phase.** |
| R2 | 🔒 | Deploy workflow interpolates `GHCR_PAT` into the SSM command text (persisted in SSM history) and uses long-lived AWS keys instead of GitHub OIDC. |
| R3 | 🧱 | No CI on pull requests and no frontend checks. Images are only tagged `:latest`, so there is no rollback. |
| R4 | 🧹 | README claims cursor pagination. There is no root Makefile and no `CLAUDE.md` conventions file. |

---

## Review 2: conventions, structure, AWS, Terraform, frontend (2026-10-10)

### Go naming & idioms (Effective Go, Go Code Review Comments, Uber Go Style Guide)

| # | Finding | Where | Fix |
|---|---|---|---|
| N1 | **Request cancellation is lost.** Handlers pass `*gin.Context` as `context.Context`; with Gin's default `ContextWithFallback=false` its `Done()` is nil, so client disconnects never cancel DB queries or Gemini calls. | all handlers | Pass `c.Request.Context()`; name the gin param `c`, not `ctx`. |
| N2 | Package names shadowed by variables: `config, err := config.LoadConfig`, `server, err := server.NewServer`, `NewServer(config config.Config…)`. | `cmd/api/main.go`, `server/server.go` | `cfg`, `srv`. |
| N3 | Stutter and layer-named packages: `usecase.AuthUsecase`, `usecase.NewAuthUsecase`, `usecase.AuthService`. One `usecase` package holds 10 services (~4k lines). | `internal/usecase` | Package by domain: `auth.Service`, `tweet.New(...)`. |
| N4 | Interfaces declared next to their implementations, plus `var _` checks. Go idiom is to define interfaces where they are *consumed* ("accept interfaces, return structs"). | `usecase/interfaces.go` | Small interfaces in `server` (the consumer); domain packages return concrete types. |
| N5 | Initialisms: `IdToken`, `AvatarUrl`, `BannerUrl`, `MediaUrl` should be `IDToken`, `AvatarURL`, `MediaURL`. Some come from sqlc. | server DTOs, `internal/db` | `rename:` in `sqlc.yaml`; fix DTO field names (JSON tags unchanged). |
| N6 | Receiver names: `server *Server` on 60 methods (should be short, e.g. `s`); `payload`/`p` and `maker` inconsistent on one type. | `server`, `token` | One short receiver per type. |
| N7 | `log.Fatal` outside `main` (`runDBMigration`) and many exit paths. Uber: "exit once, in main". | `cmd/api/main.go` | `func main() { if err := run(); err != nil {…} }`. |
| N8 | Named returns plus bare `return` (`LoadConfig`, `issueSession`); mutable package globals (`viper`, the SSM `configLoadedSuccessfully` flag, zerolog global). | `config`, `usecase/auth.go` | Plain returns; inject the logger (`*slog.Logger`). |
| N9 | `mustCurrentUserID` returns `(id, ok)`. By convention `Must…` panics. | `server/helpers.go` | `requireUserID`, or route-level middleware that guarantees it. |
| N10 | `logger` imports `middleware` (request logging lives in the logger package). | `internal/logger` | Move the request logger into `middleware`. |
| N11 | Stringly-typed enums (`NotifTypeLike = "LIKE"`, media types); config durations as `int` minutes or days. | `usecase/types.go`, `config` | `type NotificationType string`; `time.Duration` config fields. |
| N12 | Narrating comments (`// 1. …`, `// --- Section ---`); Thai comments in Dockerfile and `setup-ec2.sh`. | various | Comment only the *why*; English everywhere. |

Already idiomatic: `ErrX` sentinel naming, `errors.Is`/`errors.As`, `%w` wrapping, sqlc + pgx, `internal/` usage, small focused middleware, table-driven tests.

### Project structure

Current layout is the common "layered" one (`server → usecase → db/service`). It's fine, but the names fight Go conventions (N3), `service` actually means *AWS adapters*, and `usecase` is a grab bag. Target (package-by-feature, the layout the Go team and most Go shops use for monoliths):

```
twitter-go-api/
├── cmd/api/main.go            # wiring only: run() error
├── internal/
│   ├── config/                # env → typed Config (no viper)
│   ├── db/                    # sqlc output + Store/ExecTx, migrations/ (embedded), queries/
│   ├── auth/ user/ tweet/ feed/ search/ notification/ message/ assistant/ media/
│   │   └── service.go         # business logic; returns concrete types; domain errors
│   ├── realtime/              # SSE hub + Redis pub/sub fan-out
│   ├── storage/               # S3 presign/delete (was service/storage.go)
│   ├── embedding/             # Gemini embeddings + job worker
│   ├── apperr/
│   └── api/               # router, handlers, DTOs (types.go), middleware/
└── docs/                      # swagger (generated) + api-contract.md
```

Dependency rule: `api → feature packages → db/storage/realtime`, with shared read models in `domain`. Domain packages never import `api` or gin.

### AWS architecture

The current setup works and has good choices: OAC-protected S3 behind CloudFront, IMDSv2, EC2 Instance Connect Endpoint (no SSH keys), RDS in private subnets, SSM SecureString config, Grafana Cloud free tier. The problems:

| # | Kind | Finding |
|---|---|---|
| A1 | 🔒 | API Gateway → EC2 runs over **plain HTTP across the internet** to the instance's public IP. Port 8080 is open to `0.0.0.0/0`, so the gateway secret travels unencrypted and `/metrics`, `/healthz` and `/readyz` are publicly reachable on the instance directly. |
| A2 | 🐞 | HTTP API Gateway has a **30s integration timeout** and **no WebSocket support**. SSE notifications and assistant streams get cut, and DMs can't work. |
| A3 | 💰 | The **NAT instance exists only so the in-VPC Lambda can reach Gemini**. It costs ~$11/mo with its public IPv4 (the README's "$3.50" is wrong), and it's a hand-rolled box to patch. The Lambda also holds the DB password and Gemini key in plaintext env vars, uses a deprecated SDK, and is built by `pip` inside `terraform apply`. |
| A4 | 🔒 | RDS `storage_encrypted = false`, even though encryption is free. `gp2` should be `gp3`; `backup_retention_period = 0` should be at least 1 (free within storage size). |
| A5 | 🐞 | Prod has **no Redis**: `redis_address` defaults to `""` and the EC2 compose doesn't run it, so prod silently differs from local. That's harmless on one instance, but undocumented. |
| A6 | 🔒 | Terraform state is local and contains every secret. CI uses long-lived IAM keys, and the deploy step writes `GHCR_PAT` into the SSM command text. |

**Proposed target: simpler, cheaper (~$40 → ~$25/mo), and more secure.**

```
Browser ──► Amplify (Next.js SSR) ─┐
   │                               │ server-side fetch with cookie → bearer
   └──► api.<domain> ──► EC2 [Caddy (auto-TLS) → Go API + embedding worker, Redis, Alloy] ──► RDS (private)
                                    └──► Gemini (EC2 is in a public subnet; no NAT needed)
Media: browser ──PUT──► S3 (presigned) ──► CloudFront (OAC) ──► browser
```

- **Drop API Gateway.** Caddy on EC2 terminates TLS (Let's Encrypt). The security group opens only 80/443, and SSE/streams have no 30s cap. This fixes A1 and A2 and deletes `GatewayGuard` and the gateway secret.
- **Drop SQS, Lambda, the DLQ and the NAT instance.** Run embeddings in the Go process as a Postgres-backed job queue (`embedding_jobs` + `FOR UPDATE SKIP LOCKED` + retry/backoff). Same async lesson, works identically in local dev (fixes G21), with no Python and no plaintext secrets. Fixes A3.
- RDS: encryption on, `gp3`, 1-day backups. Redis as a container in the EC2 compose. Remote S3 state with `use_lockfile`; GitHub OIDC for CI.
- Intentional trade-offs for a learning project: single AZ, a single EC2, no load balancer.

### Terraform code quality

Good already: `default_tags`, the modern standalone `aws_vpc_security_group_*_rule` resources, SG-to-SG references, OAC instead of legacy OAI, `sensitive` variables, typed variables, and a flat root module sized for the project. The microservices repo's 17-module layout would be overkill here.

| # | Finding |
|---|---|
| T1 | `terraform fmt -check` fails (`amplify.tf`, `api_gateway.tf`, `ec2.tf`, …). There's no fmt or validate in CI; port the microservices `.githooks/pre-commit` terraform fmt hook. |
| T2 | `.terraform.lock.hcl` is **gitignored**. HashiCorp recommends committing it (reproducible provider versions). |
| T3 | Local state holding secrets. Move to an S3 backend with `use_lockfile = true` (TF ≥ 1.10, no DynamoDB needed). |
| T4 | Hardcoded values: subnet CIDRs (use `cidrsubnet(var.vpc_cidr, 8, n)`), `ap-southeast-1` inside the Amplify build spec, and `/chmtwt/prod/` repeated in TF and Go (make it one `local` / one env var). |
| T5 | Two identical private route tables (one is enough). Number-like settings are declared as `string` (`"25"`). No `validation` blocks (e.g. token key ≥ 32 chars). |
| T6 | `null_resource` + `local-exec pip install` (use `terraform_data`, or build in CI; this goes away if Lambda is dropped). The Lambda IAM role grants SSM read it never uses. |
| T7 | EC2 `user_data` nests heredocs in heredocs and installs docker compose from `releases/latest` (unpinned). Prefer a `templatefile` cloud-init with pinned versions. |
| T8 | AWS provider `~> 5.0` (v6 is current). CloudFront uses legacy `forwarded_values` (use the managed `CachingOptimized` cache policy) and the default `PriceClass_All` (use `PriceClass_200` for Asia, or `100`). |

### Frontend: microservices vs. monolith, verdict per area

**Port from microservices (clearly better):**
- **Design tokens:** exact X palette (`#1D9BF0`, `#536471`, `#2F3336`…), semantic `--like` / `--retweet` / `--banner` tokens, and a z-index scale. The monolith hardcodes `text-red-500` / `text-pink-600` / `text-green-500` and inconsistent dark greys.
- **Component design:** `Avatar` (deterministic colored-initials fallback), `TweetActionButton` (one source for hover/active colors and counts), `TweetActionsMenu`, `UserRow` with `default|compact` variants and skeleton, `ConfirmModal` (delete/logout), `ImageLightbox`, `Tabs` primitive, `BackButton`. The monolith repeats this markup inline (three copies of the nav-item JSX in `AppNav`; avatar fallback everywhere; hand-built tab bars on four pages).
- **States:** every list has skeleton, error + retry, and empty copy. The monolith shows `"Loading..."` text and no error states.
- **URL-driven UI:** profile tabs as routes (`/[username]/replies|media|likes`); followers/following as pages instead of modals; explore query synced to `?q=`.
- **Composer:** char-count ring, emoji picker, expand on focus, client-side size check, preview URL revoke on unmount.
- **Edit profile:** floating-label fields, counters shown on focus, location + website, dirty-check on the Save button.
- **Tweet detail:** a distinct large layout (23px body, full timestamp, stats row).
- **Notifications:** per-item mark-read plus a "Mark all as read" button. The monolith marks everything read on page mount, so the unread highlight disappears instantly.
- **A11y:** `aria-label`s on icon buttons. The monolith uses `<div onClick>` for back buttons.
- **Data layer:** `query-keys.ts`, `debounced-toggle.ts`, `applyTweetUpdate` / `removeTweetEverywhere`, single-flight refresh, SSR prefetch with `HydrationBoundary`, `loading.tsx` per route.

**Keep from the monolith (it's better or more complete here):**
- Mobile bottom nav (the microservices app has none below `sm`), public browsing for logged-out users with a sign-in modal (microservices walls everything behind login), and `/{username}` URLs (microservices uses opaque `/profile/{uuid}`).
- Working DMs and ChanomBot (microservices messages and bookmarks are placeholders), the theme toggle, notification context previews (reply plus original tweet), and video support.
- Radix-based shadcn primitives and `sonner` toasts. Porting the *designs* doesn't require switching to Base UI; swapping primitive libraries is churn with no user-visible gain.
- ESLint (`eslint-config-next` covers React Hooks + Next rules). Biome is optional.

**Don't copy:** fake affordances (Bookmark and analytics buttons that do nothing, dead footer links), the "Not available in this build" bookmarks page, and `normalize.ts` snake→camel mapping (the monolith API is already camelCase).

---

## Microservices repo: what to port vs. leave

**Port (fits a monolith):**
- Next.js **BFF auth**: httpOnly cookies set by route handlers, `proxy.ts` for silent refresh, and a server-side fetch helper, plus **SSR prefetch with `HydrationBoundary`** and `loading.tsx` skeletons.
- `lib/query-keys.ts` (single source of truth plus matchers), `lib/debounced-toggle.ts` (module-level, state-diffing), single `applyTweetUpdate` / `removeTweetEverywhere` cache helpers, and an axios single-flight refresh.
- UI: `Avatar` (deterministic color fallback), `TweetActionButton`, `ConfirmModal`, `ImageLightbox`, composer with char-count ring and emoji picker, profile layout with URL-driven tabs (Posts / Replies / Media grid / Likes), location/website/joined meta, onboarding (choose username), `formatCount` / `displayNameOf` / `renderTweetBody`, Twitter-exact oklch theme tokens (`--like`, `--retweet`) and z-index scale, `useInfiniteScroll`.
- Go: `slog` logger, a `healthz` helper with per-dependency status, the `pgxutil.MapErr` idea (map DB errors at the boundary), `signal.NotifyContext`, `types.go` convention, keyset pagination, separate `retweets` table and feed items with `kind`.
- Tooling: PR CI workflow, GitHub OIDC for AWS, `CLAUDE.md` conventions (adapted), Biome (optional), Playwright smoke test (optional).

**Leave behind (distributed-only complexity):** Kafka/outbox, gRPC, circuit breakers, Kong, Keycloak, OpenSearch, EKS, hybrid fan-out, Redis snapshot denormalization, OTel tracing.

**Verdict on deleting it:** fine to delete locally. Recommend **archiving the GitHub repo** (read-only) instead of deleting it: zero maintenance, and the transactional-outbox, DLQ and fan-out code stays available as a reference if you revisit distributed systems.

---

## Decisions
2026-10-10, round 1:
- Java API: **keep**; improve it last (Phase 7), after the Go API and web are done.
- Migrations: **squash**; local and prod data are disposable.
- Auth: **httpOnly cookies** with SSR prefetch (design refined below).
- Brainrot / premium / TikTok pages: **keep**.

2026-10-10, round 2:
- Go API: **package-by-feature** layout (see "Project structure"), done in Phase 2.
- AWS: **replace API Gateway with Caddy TLS on EC2**; **replace SQS + Lambda + NAT with a Postgres job queue** in Go; **harden** RDS, state and CI.
- Domain: `api.chanombude.me` (Go API on EC2) and `twitter.chanombude.me` (Amplify).
- Auth design, refined by having a domain: the **Go API sets httpOnly cookies** (`Domain=.chanombude.me`, `SameSite=Lax`, `Secure`). The browser calls `api.*` directly with `credentials: "include"` (SSE included, so no Next proxy hop). Next server components read the cookie and forward it as `Authorization: Bearer`. CSRF: `SameSite=Lax`, a strict CORS allowlist, and an `Origin` check on unsafe methods. Locally, `localhost:3000` and `localhost:8080` are same-site, so this works unchanged.

## Phases

### Phase 0: Repo hygiene
- [x] Add PR CI: Go vet/test with `-race`; web lint, typecheck and build
- [ ] Add `CLAUDE.md` with the conventions below (after Phase 2 settles them)
- (Java leftovers `.github/modernize`, `twitter-java-api/.idea` → Phase 7)

### Phase 1: Backend correctness & security ✅
- [x] G1/G2 object keys `{folder}/{userId}/{uuid}{ext}`, validated server-side; URLs rejected; size signed into the presign; only delete keys from our own bucket
- [x] G3 email only in `/auth/me`
- [x] G4 require `email_verified`
- [x] G6 cosine operator `<=>`
- [x] G7 SSE framing for the assistant (and client parser)
- [x] G11 make init migration non-destructive (squashed into a single `000001_init_schema`; **existing DBs must be reset**)
- [x] G12 dedupe notifications
- [x] G17 single Gemini client
- [x] G20 `go mod tidy`, bump Go/Gin, pin Docker base
- [x] Frontend quick fixes W1, W2, W3, W5 (plus: stale `['auth','user']` key after a profile edit, obsolete `postinstall` patch removed)

### Phase 2: Backend foundation
- [x] **2a Structure & idioms:** package-by-feature move (N3, N4, N10); `run() error` main (N7); `c.Request.Context()` everywhere (N1); naming fixes N2, N5, N6, N9; embed migrations; per-package `Config` with `time.Duration` TTLs (N11, partly; the rest lands in 2b). `internal/db` kept its name (it's the sqlc convention); query strings are no longer logged (G5, partly)
- [x] **2b Platform:** `slog` with request-ID-aware handler; env config via `caarlos0/env` + optional `app.env` (no viper, no in-app SSM fetch; typed durations, `notEmpty` checks, all errors reported at once) (G15, G16, N8, N11); unified error mapping in `httpError` + `db.NotFound` for resource-specific 404s (G14); `/readyz` reports each dependency without leaking error details; sanitized `X-Request-ID`
  - ⚠️ Env vars renamed: `FRONTEND_URL`→`ALLOWED_ORIGINS`, `TOKEN_DURATION_MINUTES`→`ACCESS_TOKEN_TTL` (`15m`), `REFRESH_TOKEN_DURATION_DAYS`→`REFRESH_TOKEN_TTL` (`720h`), `DB_MAX_CONN_LIFETIME_MINUTES`→`DB_MAX_CONN_LIFETIME` (`5m`), `REDIS_ADDRESS`+`REDIS_PASSWORD`→`REDIS_URL`, `S3_BUCKET_NAME` unchanged. The app no longer reads SSM itself: Phase 5 must render SSM into the container's env (and rename the SSM keys).
- [x] **2c Realtime:** `internal/realtime` hub; **one SSE stream** at `GET /api/v1/events` (`notification` + `dm` events) fanned out via Redis; WebSocket handler, its connection registry and the token-in-query-string removed (G10, G13, G5). Frontend: `useRealtimeEvents` replaces `useNotificationSSE` + `useChatWebSocket`
- [x] **2d Data:** opaque keyset cursors (`internal/pagination`) for every chronological list, an anchored `tweet_score()` cursor for For-You, offset-in-cursor for relevance search (G9); `retweets` table, with retweets returned as the original tweet + `retweetedBy`/`retweetedAt` instead of wrapper tweets (G19); per-device refresh tokens rotated via a single `DELETE … RETURNING` (G8). `dbtest.MockStore` is now generated from `db.Querier`; 12 Postgres integration tests (run in CI via a service container)
- [x] **2e Features:** `GET /users/by-username/:username`; `GET /users/:id/{posts,replies,media,likes}` (likes owner-only); `location`/`website`/`createdAt` on users; case-insensitive usernames + `POST /users/me/onboarding` (G18)
- [x] **2f Auth cookies:** login/refresh set httpOnly `access_token` + `refresh_token` cookies (`SameSite=Lax`, `Secure` in prod, `COOKIE_DOMAIN`), logout clears them; tokens no longer in response bodies; middleware accepts cookie or Bearer; `http.CrossOriginProtection` refuses cross-origin writes; a failed refresh never clears cookies (concurrent-refresh race)
- [x] **2g Embeddings:** `embedding_jobs` enqueued in the tweet's own transaction; in-process worker leases jobs (`FOR UPDATE SKIP LOCKED`, 5 min lease), retries with exponential backoff and gives up after 8 attempts; the assistant embeds questions through the same `embedding.Embedder`. SQS publisher removed; works locally (G21). The SQS/Lambda/NAT Terraform and `infra/lambda` are removed in Phase 5

### Phase 3: Frontend architecture
- [x] Cookie auth against the API (no tokens in JS, W6): `proxy.ts` renews expired sessions before rendering; root layout seeds the current user from the server (no `AuthInitializer` spinner); `useAuth` reads it from TanStack Query; single-flight refresh on `TOKEN_EXPIRED` (new API code, also from optional-auth routes so signed-in users never get silent anonymous data); native `EventSource` with credentials (polyfill removed); per-tab `QueryClient` (W8)
- [ ] SSR prefetch + `HydrationBoundary` for the main pages; `loading.tsx`
- [x] `lib/query-keys.ts` (every key, W3), `lib/query-cache.ts` (one shape-aware updater for every cached copy of a tweet/user, incl. tweet authors and search results, which were previously missed), module-level `lib/debounced-toggle.ts` for like/retweet/follow (W7: survives unmount, one timer per tweet, no request if toggled back), single-flight refresh, one SSE hook; Vitest for `lib/` in CI
- [ ] W8/W9 cleanup (keep the brainrot/premium pages); split messages page

### Phase 4: Frontend UI port (see "Frontend" verdicts)
- [ ] Theme tokens + z-index scale; `Avatar`, `TweetActionButton`, `TweetActionsMenu`, `UserRow`, `ConfirmModal`, `ImageLightbox`, `Tabs`, `BackButton`, skeletons
- [ ] Composer (ring, emoji, focus expand); tweet detail layout; edit profile (floating fields, location/website)
- [ ] Profile route tabs (`/[username]/replies|media|likes`), followers/following pages, onboarding, explore `?q=` sync, notifications mark-read UX
- [ ] Keep: mobile nav, public browsing + sign-in modal, `/{username}` URLs, DMs, ChanomBot, theme toggle

### Phase 5: Infrastructure (Terraform) ✅
- [x] Removed API Gateway, SQS/DLQ, Lambda, NAT instance, `infra/lambda`, `GatewayGuard`/`GATEWAY_SECRET`. EC2 SG allows 80/443 (+ SSH from the EICE only); Caddy + Redis in `infra/ec2/compose.yml`; pinned image versions; Elastic IP
- [x] Secrets out of user data: cloud-init only writes files; `deploy.sh` renders `api.env` from SSM (`/<project>/<env>/api/*`, one parameter per env var, empty values skipped)
- [x] RDS encrypted, `gp3`, 1-day backups, Postgres 16; RDS reachable only from the API host; private subnets have no internet route
- [x] S3 remote state with `use_lockfile` (`infra/bootstrap` creates the versioned bucket); `.terraform.lock.hcl` committed; AWS provider `~> 6.0`, Terraform `>= 1.10`
- [x] T1, T4–T8: `cidrsubnet`, one `local.name` prefix, `number`/`bool` types, input validations, managed CloudFront cache policy + `PriceClass_200`, multipart-abort lifecycle, PUT-only bucket CORS
- [x] Amplify custom domain `twitter.<domain>`, env vars set directly (no SSM read at build); optional Route 53 record for `api.<domain>`, otherwise `dns_records` output
- [x] `terraform test` plans the stack against a mocked AWS provider (no credentials)

### Phase 6: CI/CD & docs
- [x] GitHub OIDC deploy role (branch-scoped); ECR with immutable SHA tags + lifecycle; no secrets in SSM command text; rollback via `image_tag` input; post-deploy `/readyz` check
- [x] `terraform fmt -check` / `validate` / `test` in CI; README AWS sections rewritten
- [ ] Pre-commit hook; architecture diagram image; `CLAUDE.md`

### Phase 7: Java API parity
- [ ] Clean `.idea` and `.github/modernize`; align `twitter-java-api` with the squashed schema and the Go API contract (DMs, assistant, presign, cursors, cookie auth)
