# Twitter Go API Contract (Frontend)

Base URL: `/api/v1`

## Auth & Session
- Sessions live in two **httpOnly cookies** set by the API: `access_token` (short-lived JWT) and `refresh_token` (rotating, single use). Page scripts never see the tokens.
- Browsers send them automatically: call the API with credentials (`fetch(..., { credentials: "include" })`, axios `withCredentials: true`, `new EventSource(url, { withCredentials: true })`).
- Cookies are `HttpOnly; SameSite=Lax; Path=/`, `Secure` in production, host-only by default (production serves the web app and the API on one domain; in dev, `localhost` covers every port). `COOKIE_DOMAIN` scopes them to a parent domain if the two are ever split across subdomains.
- `401` comes with one of two codes:
  - `UNAUTHORIZED`: no credentials were sent. The user is signed out; show sign-in.
  - `TOKEN_EXPIRED`: credentials were sent but are stale (expired access token, or only the refresh cookie is left). Call `POST /auth/refresh` and retry. Share one in-flight refresh between concurrent requests: each refresh token works once. Optional-auth endpoints return this too, so a signed-in user never silently gets anonymous data.
- Unsafe requests (`POST`/`PUT`/`PATCH`/`DELETE`) from origins outside `ALLOWED_ORIGINS` are refused with `403` (CSRF protection).
- Non-browser clients (curl, tests) may send `Authorization: Bearer <access token>` instead; the header wins over the cookie.

## Common Query Params
- Pagination (where supported):
  1. `cursor`: opaque token from the previous response's `nextCursor`. Don't parse or build it.
  2. `size`: default `20`, max `50`.
- Chronological lists use keyset cursors (the sort key of the last item seen), so new posts never shift or duplicate items between pages. The ranked "For You" feed fixes its ranking time on the first page. Relevance-ranked search pages by offset.

## PageResponse<T>
```json
{
  "items": [],
  "hasNext": true,
  "nextCursor": "MTAw"
}
```

## Common Error Shape
```json
{
  "code": "BAD_REQUEST|UNAUTHORIZED|TOKEN_EXPIRED|FORBIDDEN|NOT_FOUND|CONFLICT|INTERNAL_ERROR|VALIDATION_ERROR|TOO_MANY_REQUESTS",
  "message": "string",
  "details": [
    { "field": "string", "message": "string" }
  ]
}
```

## Data Models

### UserResponse
```json
{
  "id": 1,
  "username": "alice",
  "displayName": "Alice",
  "bio": "hello",
  "location": "Bangkok",
  "website": "https://example.com",
  "avatarUrl": "https://d1234.cloudfront.net/avatars/1/0b1c....jpg",
  "bannerUrl": null,
  "isFollowing": false,
  "followersCount": 10,
  "followingCount": 20,
  "createdAt": "2026-03-01T00:00:00Z"
}
```

### TweetResponse
```json
{
  "id": 1,
  "content": "tweet text",
  "mediaType": "IMAGE|VIDEO",
  "mediaUrl": "https://d1234.cloudfront.net/tweets/uuid_photo.jpg",
  "user": { "...UserResponse" },
  "replyCount": 0,
  "likeCount": 0,
  "retweetCount": 0,
  "isLiked": false,
  "isRetweeted": false,
  "replyToTweetId": 123,
  "replyToUsername": "alice",
  "createdAt": "2026-03-02T00:00:00Z",
  "retweetedBy": null,
  "retweetedAt": null
}
```
- In timelines (following feed, profile posts) a retweet appears as the **original tweet** with `retweetedBy` (UserResponse) and `retweetedAt` set: "bob reposted". The same tweet can therefore appear twice in a list (as posted and as retweeted), so key list items by `retweetedBy.id` + `id`.

### HashtagResponse
```json
{
  "id": 1,
  "text": "golang",
  "usageCount": 12,
  "lastUsedAt": "2026-03-02T00:00:00Z",
  "createdAt": "2026-03-01T00:00:00Z"
}
```

### NotificationResponse
```json
{
  "id": 1,
  "actor": { "...UserResponse" },
  "tweetId": 10,
  "tweetContent": "tweet text",
  "tweetMediaUrl": "https://...",
  "originalTweetId": 9,
  "originalTweetContent": "parent tweet",
  "originalTweetMediaUrl": "https://...",
  "type": "LIKE|REPLY|RETWEET|FOLLOW",
  "isRead": false,
  "createdAt": "2026-03-02T00:00:00Z"
}
```

## Endpoints

## Auth

### POST `/auth/google`
Body:
```json
{ "idToken": "google_id_token" }
```
Response 200: `CurrentUserResponse`, and `Set-Cookie` for both session cookies.

### POST `/auth/refresh`
No body; reads the `refresh_token` cookie.
Response 200: `CurrentUserResponse`, and `Set-Cookie` with a **new** pair. The old refresh token stops working (rotation).
Response 401 when the cookie is missing, used, expired or revoked. Cookies are left untouched on failure (a concurrent refresh may have just renewed them).

### POST `/auth/logout`
No body; reads the `refresh_token` cookie.
- Ends this browser's session and deletes both cookies. Other devices stay signed in.
Response 200:
```json
{ "success": true }
```

### GET `/auth/me` (private)
Response 200: `CurrentUserResponse`: `UserResponse` plus `"email"` and `"onboarded"` (false until the user has chosen a username). Email is never included in public `UserResponse`s.

## Uploads

### POST `/uploads/presign` (private)
Request a presigned S3 PUT URL. The client then PUTs the file directly to S3.

Body:
```json
{
  "contentType": "image/jpeg",
  "folder": "tweets|avatars|banners",
  "contentLength": 204800
}
```
Response 200:
```json
{
  "presignedUrl": "https://bucket.s3.amazonaws.com/...",
  "objectKey": "tweets/42/0b1c4f9e-....jpg"
}
```

- Allowed content types: `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `video/mp4`, `video/webm` (`avatars`/`banners` accept images only).
- `contentLength` is required and checked against the folder's size limit. Content type and length are signed into the URL, so the `PUT` must send exactly those `Content-Type` and `Content-Length` values.
- Object keys are always `{folder}/{userId}/{uuid}{ext}`. Endpoints that accept a key reject keys belonging to another user or folder.

## Users

### GET `/users/:id` (optional auth)
Response 200: `UserResponse`

### GET `/users/by-username/:username` (optional auth)
Case-insensitive lookup. Response 200: `UserResponse`; 404 when no such user.

### POST `/users/me/onboarding` (private)
Body:
```json
{ "username": "alice_99", "displayName": "Alice" }
```
- `username`: 3-20 letters, numbers or underscores; unique regardless of case (409 `CONFLICT` when taken).
Response 200: `CurrentUserResponse` (with `onboarded: true`)

### Profile tabs (optional auth)
All take `cursor`, `size` and return `PageResponse<TweetResponse>`, newest first.
- GET `/users/:id/posts`: top-level tweets and retweets
- GET `/users/:id/replies`: replies the user wrote
- GET `/users/:id/media`: tweets with an image or video
- GET `/users/:id/likes` (private): tweets the user liked; 403 for anyone but the user themselves

### PUT `/users/profile` (private)
Body:
```json
{
  "displayName": "Alice",
  "bio": "hello world",
  "location": "Bangkok",
  "website": "https://example.com",
  "avatarKey": "avatars/42/0b1c....jpg",
  "bannerKey": ""
}
```
- Every field is optional: omit to keep the current value, `""` to clear it. `website` must be an http(s) URL.
- `avatarKey` / `bannerKey`: omit to keep the current image, `""` to remove it, or an object key from the presign endpoint (must be in `avatars/{yourId}/` or `banners/{yourId}/`).

Response 200: `UserResponse`

## Tweets

### POST `/tweets` (private)
Body:
```json
{
  "content": "tweet text",
  "parentId": 123,
  "mediaKey": "tweets/42/0b1c....jpg"
}
```
- `content`: optional (max 280), required when `mediaKey` is not provided
- `parentId`: optional (reply to tweet)
- `mediaKey`: optional object key from the presign endpoint (must be in `tweets/{yourId}/`). The response's `mediaType` (`IMAGE`/`VIDEO`) is derived from it.

Response 201: `TweetResponse`

### DELETE `/tweets/:id` (private)
Response 200:
```json
{ "success": true }
```

### GET `/tweets/:id` (optional auth)
Response 200: `TweetResponse`

### GET `/tweets/:id/replies` (optional auth)
Query: `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

### POST `/tweets/:id/like` (private)
Response 200:
```json
{ "success": true }
```

### DELETE `/tweets/:id/like` (private)
Response 200:
```json
{ "success": true }
```

### POST `/tweets/:id/retweet` (private)
Idempotent. Response 200: the retweeted `TweetResponse` (with `isRetweeted: true`).

### DELETE `/tweets/:id/retweet` (private)
Response 200:
```json
{ "success": true }
```

## Feeds

### GET `/feeds/global` (optional auth)
Query: `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

### GET `/feeds/following` (private)
Tweets and retweets by accounts you follow, newest first.
Query: `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

## Search

### GET `/search/users` (optional auth)
Query:
1. `q` (required)
2. `cursor`, `size`
Response 200: `PageResponse<UserResponse>`

### GET `/search/tweets` (optional auth)
Query:
1. `q` (required)
2. `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

### GET `/search/hashtags` (optional auth)
Query:
1. `q` (optional, empty -> `[]`)
2. `limit` (default `5`, max `50`)
Response 200: `HashtagResponse[]`

## Discovery

### GET `/discovery/trending` (optional auth)
Query:
1. `limit` (default `10`, max `50`)
Response 200: `HashtagResponse[]`

### GET `/discovery/users` (optional auth)
Query: `cursor`, `size`
Response 200: `PageResponse<UserResponse>`

## Notifications

### GET `/notifications` (private)
Query: `cursor`, `size`
Response 200: `PageResponse<NotificationResponse>`

### GET `/notifications/unread-count` (private)
Response 200: number

### POST `/notifications/mark-read` (private)
Response 200:
```json
{ "success": true }
```

## Real-time Events

### GET `/events` (private, SSE)
One stream per signed-in tab carries every real-time update. Events reach the user on whichever API instance their stream is connected to (Redis pub/sub).
- Content-Type: `text/event-stream`; the server suggests `retry: 3000`.
- Events:
  1. `connected`: `{ "status": "ok" }`. Sent on every (re)connect; refetch anything you may have missed.
  2. `notification`: `NotificationResponse`
  3. `dm`: `MessageResponse`. Sent to every participant, including the sender's other tabs.
- Comment lines (`: ping`) every 15s are heartbeats.

## Direct Messages

### GET `/messages/conversations` (private)
Query: `cursor`, `size`
Response 200: `PageResponse<ConversationResponse>`

### GET `/messages/conversations/:id/messages` (private)
Query: `cursor`, `size`
Response 200: `PageResponse<MessageResponse>`

### POST `/messages/conversations/:id/messages` (private)
Body:
```json
{
  "content": "Message content"
}
```
Response 200: `MessageResponse`

### POST `/messages/users/:id/messages` (private)
Starts a new conversation (or adds to an existing conversation) with a user by ID.
Body:
```json
{
  "content": "Message content"
}
```
Response 200: `MessageResponse`

## AI Assistant

### POST `/assistant` (private, SSE)
- Content-Type: `application/json`
Body:
```json
{
  "query": "User query here",
  "history": [
    { "role": "user", "text": "Previous query" },
    { "role": "model", "text": "Previous reply" }
  ]
}
```
- Stream response over Server-Sent Events (SSE). Every `data:` payload is JSON, so newlines in the model output never break framing:
  1. default event: `data: {"text": "partial chunk"}` (repeated)
  2. `event: done` with `data: {}` (stream finished)
  3. `event: error` with `data: {"message": "..."}` (stream aborted)
