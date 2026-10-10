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

## Phases

## Decisions (2026-10-10)
- Java API: **keep**; improve it last (Phase 7), after the Go API and web are done.
- Migrations: **squash**; local and prod data are disposable.
- Auth: **httpOnly-cookie BFF** in Next.js with SSR prefetch; the Go API stays bearer-token based.
- Brainrot / premium / TikTok pages: **keep**.

### Phase 0: Repo hygiene
- [ ] Remove `.github/modernize` leftovers and committed `twitter-java-api/.idea`
- [ ] Add `CLAUDE.md` with conventions
- [x] Add PR CI: Go vet/test with `-race`; web lint, typecheck and build

### Phase 1: Backend correctness & security (no API shape changes)
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
- [ ] G15/G16 `slog` and simplified env config
- [ ] G14 unified error mapping
- [ ] G13 extract `internal/realtime` hub; G10/G5 **one SSE stream** for notifications and DMs via Redis pub/sub (drop gorilla/websocket)
- [ ] G9 keyset cursors (chronological lists) and an anchored-time cursor for For-You
- [ ] G8 per-session refresh-token rotation
- [ ] G18 `GET /users/by-username/:username`, user likes/replies/media endpoints, user `location`/`website`, username onboarding
- [ ] G19 separate `retweets` table and `kind` feed items

### Phase 3: Frontend architecture
- [ ] W6 cookie BFF auth, `proxy.ts`, SSR prefetch, `loading.tsx`
- [ ] Centralized query keys, cache helpers, debounced toggle, single-flight refresh
- [ ] W8/W9 cleanup (keep the brainrot/premium pages); split messages page

### Phase 4: Frontend UI port
- [ ] Theme tokens, Avatar, TweetCard/ActionButton, ConfirmModal, Lightbox, Composer
- [ ] Profile layout and tabs, onboarding, trending sidebar, skeletons

### Phase 5: AI / RAG
- [ ] G21 in-process embedder fallback for local dev; Lambda migrated to `google-genai`

### Phase 6: CI/CD & docs
- [ ] R2 OIDC and SHA-tagged images; no secrets in SSM command text
- [ ] README and api-contract refresh

### Phase 7: Java API parity
- [ ] Align `twitter-java-api` with the squashed schema and the Go API contract (DMs, assistant, presign, cursors, cookie-BFF-compatible auth)
