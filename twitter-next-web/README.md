# Twitter Next Web

Frontend for the Twitter clone, built with Next.js App Router.

## Stack

- Next.js 16 (App Router), React 19, TypeScript
- Tailwind CSS v4, Radix / shadcn primitives
- TanStack Query, Zustand (UI state only), Zod
- Vitest

## API Expectations

This frontend talks to the Go API (or the Java one; same contract), described in
[`../twitter-go-api/docs/api-contract.md`](../twitter-go-api/docs/api-contract.md):

- Lists page with `cursor` + `size` and return `{ items, hasNext, nextCursor }` (`nextCursor` is `null` on the last page).
- Timelines (feeds, profile tabs) return `{ kind, tweet, retweet }` feed items; search and thread replies return plain tweets.

## Local Setup

### Prerequisites

- Node.js 22+
- npm
- The API running on `http://localhost:8080` (see the root README)

### Install + Run

```bash
npm install
npm run dev
```

Open: `http://localhost:3000`

### Environment

```bash
cp .example.env .env.local   # then set NEXT_PUBLIC_GOOGLE_CLIENT_ID
```

| Variable | Purpose |
|---|---|
| `NEXT_PUBLIC_API_URL` | API base URL for the browser (`http://localhost:8080/api/v1` locally, `/api/v1` in production) |
| `NEXT_PUBLIC_GOOGLE_CLIENT_ID` | Google OAuth client ID for sign-in |
| `API_INTERNAL_URL` (optional) | API base URL for server rendering; defaults to `NEXT_PUBLIC_API_URL` |

`NEXT_PUBLIC_*` values are compiled into the browser bundle at build time.

## Quality Checks

The same checks CI runs:

```bash
npm run lint && npx tsc --noEmit && npm test && npm run build
```

## Auth

The Go API keeps the session in two **httpOnly cookies** (`access_token`, `refresh_token`); this app never sees or stores tokens.

- **First render:** `src/proxy.ts` renews an expired access cookie before the page renders, and the root layout fetches the signed-in user with the request's cookies (`src/api/server.ts`). The user is known on the first paint, with no auth spinner.
- **In the browser:** the user lives in TanStack Query (`queryKeys.auth.me`, read with `useAuth()`). `src/api/axiosInstance.ts` sends cookies (`withCredentials`), and on a `TOKEN_EXPIRED` response renews the session once (shared by concurrent requests) and retries.
- **Realtime:** one native `EventSource` (with credentials) for notifications and DMs, `src/hooks/useRealtimeEvents.ts`.
- **Local dev:** the web app (`localhost:3000`) and the API (`localhost:8080`) are the same site, and a `localhost` cookie reaches every port, so no extra setup is needed. In production nginx serves both on one domain: the browser calls the relative `/api/v1`, and the Next.js server calls the API directly over the Docker network (`API_INTERNAL_URL`).

## Data fetching

- Query keys: all in `src/lib/query-keys.ts`; never inline arrays.
- Optimistic updates: `src/lib/query-cache.ts` updates every cached copy of a tweet or user (feeds, profile tabs, search, details, tweet authors and retweeters), whether it's cached as a plain tweet or inside a feed item.
- Like / retweet / follow: the cache flips on click and `src/lib/debounced-toggle.ts` sends one request for the final state (none if toggled back), even if the component unmounts.

## Main Directories

- `src/app`: routes/layouts
- `src/api`: API clients (browser: `axiosInstance.ts`; server render: `server.ts`)
- `src/components`: UI components
- `src/hooks`: data hooks and client logic
- `src/lib`: query keys, utilities, validation
- `src/store`: Zustand stores (UI state only)
- `src/types`: shared TypeScript contracts
