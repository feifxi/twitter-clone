# Twitter Next Web

Frontend for the Twitter clone, built with Next.js App Router.

## Stack

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS
- TanStack Query
- Zustand
- Zod

## API Expectations

This frontend expects the Go API (`/api/v1`) and cursor-based pagination:

- Request params: `cursor`, `size`
- Response model:
  - `items`
  - `hasNext`
  - `nextCursor`

## Local Setup

### Prerequisites

- Node.js 20+
- npm

### Install + Run

```bash
npm install
npm run dev
```

Open: `http://localhost:3000`

### Environment

Create `.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your-google-client-id
```

## Quality Checks

```bash
npx tsc --noEmit
npm run lint
```

## Auth

The Go API keeps the session in two **httpOnly cookies** (`access_token`, `refresh_token`); this app never sees or stores tokens.

- **First render:** `src/proxy.ts` renews an expired access cookie before the page renders, and the root layout fetches the signed-in user with the request's cookies (`src/api/server.ts`). The user is known on the first paint, with no auth spinner.
- **In the browser:** the user lives in TanStack Query (`queryKeys.auth.me`, read with `useAuth()`). `src/api/axiosInstance.ts` sends cookies (`withCredentials`), and on a `TOKEN_EXPIRED` response renews the session once (shared by concurrent requests) and retries.
- **Realtime:** one native `EventSource` (with credentials) for notifications and DMs, `src/hooks/useRealtimeEvents.ts`.
- **Local dev:** the web app (`localhost:3000`) and the API (`localhost:8080`) are the same site, and a `localhost` cookie reaches every port, so no extra setup is needed. In production both run on subdomains of one domain and the API sets `COOKIE_DOMAIN`.

## Data fetching

- Query keys: all in `src/lib/query-keys.ts`; never inline arrays.
- Optimistic updates: `src/lib/query-cache.ts` updates every cached copy of a tweet or user (feeds, profile tabs, search, details, tweet authors).
- Like / retweet / follow: the cache flips on click and `src/lib/debounced-toggle.ts` sends one request for the final state (none if toggled back), even if the component unmounts.

## Tests

```bash
npm test   # Vitest: src/**/*.test.ts
```

## Main Directories

- `src/app`: routes/layouts
- `src/api`: API clients (browser: `axiosInstance.ts`; server render: `server.ts`)
- `src/components`: UI components
- `src/hooks`: data hooks and client logic
- `src/lib`: query keys, utilities, validation
- `src/store`: Zustand stores (UI state only)
- `src/types`: shared TypeScript contracts
