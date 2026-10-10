# Twitter Go API Contract (Frontend)

Base URL: `/api/v1`

## Auth & Session
- Authentication is strictly token-based. No cookies are used.
- The frontend must store tokens in **LocalStorage**.
- All private requests must include the header: `Authorization: Bearer <accessToken>`.
- The `refreshToken` is used via JSON body to obtain new access tokens.

## Common Query Params
- Pagination (where supported):
  1. `cursor` (opaque token from previous response `nextCursor`)
  2. `size` (default `20`, max `50`)

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
  "code": "BAD_REQUEST|UNAUTHORIZED|FORBIDDEN|NOT_FOUND|CONFLICT|INTERNAL_ERROR|VALIDATION_ERROR|TOO_MANY_REQUESTS",
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
  "avatarUrl": "https://d1234.cloudfront.net/avatars/1/0b1c....jpg",
  "isFollowing": false,
  "followersCount": 10,
  "followingCount": 20
}
```

### AuthResponse
```json
{
  "accessToken": "jwt",
  "refreshToken": "uuid/jwt",
  "user": { "...UserResponse" }
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
  "retweetedTweet": { "...TweetResponse" },
  "replyToTweetId": 123,
  "replyToUsername": "alice",
  "createdAt": "2026-03-02T00:00:00Z"
}
```

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
Response 200: `AuthResponse`

### POST `/auth/refresh`
Body:
```json
{ "refreshToken": "string" }
```
Response 200: `AuthResponse`

### POST `/auth/logout`
Body (Optional):
```json
{ "refreshToken": "string" }
```
- If `Authorization` header is present, it revokes the session associated with the access token.
- If `refreshToken` is provided in the body, it explicitly revokes that refresh token.
Response 200:
```json
{ "success": true }
```

### GET `/auth/me` (private)
Response 200: `CurrentUserResponse`: `UserResponse` plus `"email"`. Email is never included in public `UserResponse`s; `AuthResponse.user` is also a `CurrentUserResponse`.

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

### PUT `/users/profile` (private)
Body:
```json
{
  "displayName": "Alice",
  "bio": "hello world",
  "avatarKey": "avatars/42/0b1c....jpg",
  "bannerKey": ""
}
```
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
Response 200: `TweetResponse`

### DELETE `/tweets/:id/retweet` (private)
Response 200:
```json
{ "success": true }
```

## Feeds

### GET `/feeds/global` (optional auth)
Query: `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

### GET `/feeds/user/:id` (optional auth)
Query: `cursor`, `size`
Response 200: `PageResponse<TweetResponse>`

### GET `/feeds/following` (private)
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
