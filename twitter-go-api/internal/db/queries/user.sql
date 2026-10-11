-- name: CreateUser :one
INSERT INTO users (username, email, display_name, bio, avatar_url, banner_url, role, provider)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUser :one
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM users u
WHERE u.id = sqlc.arg('id');

-- name: GetUserByUsername :one
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM users u
WHERE lower(u.username) = lower(sqlc.arg('username'));

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET bio = $2,
    display_name = $3,
    avatar_url = $4,
    banner_url = $5,
    location = $6,
    website = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUsername :one
-- Marks onboarding complete. A taken username fails on idx_users_username_lower.
UPDATE users
SET username = $2,
    display_name = $3,
    onboarded_at = COALESCE(onboarded_at, now()),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetUsersByIDs :many
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM users u
WHERE u.id = ANY(sqlc.arg('user_ids')::bigint[]);

-- name: ListFollowers :many
-- Accounts following @user_id, most recent follow first; the cursor is the follow's time.
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f2 WHERE f2.following_id = u.id AND f2.follower_id = sqlc.narg('viewer_id')) AS is_following,
  f.created_at AS followed_at
FROM follows f
JOIN users u ON u.id = f.follower_id
WHERE f.following_id = sqlc.arg('user_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (f.created_at, f.follower_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY f.created_at DESC, f.follower_id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListFollowing :many
-- Accounts @user_id follows, most recent follow first.
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f2 WHERE f2.following_id = u.id AND f2.follower_id = sqlc.narg('viewer_id')) AS is_following,
  f.created_at AS followed_at
FROM follows f
JOIN users u ON u.id = f.following_id
WHERE f.follower_id = sqlc.arg('user_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (f.created_at, f.following_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY f.created_at DESC, f.following_id DESC
LIMIT sqlc.arg('page_limit');

-- name: SearchUsers :many
-- Ranked by popularity, so it pages by offset.
SELECT sqlc.embed(u),
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM users u
WHERE u.username ILIKE '%' || sqlc.arg('query') || '%'
   OR u.display_name ILIKE '%' || sqlc.arg('query') || '%'
ORDER BY u.followers_count DESC, u.id DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: ListSuggestedUsers :many
-- Most-followed accounts the viewer doesn't follow yet (anyone, for anonymous viewers).
-- Accounts still on their generated handle (not onboarded) aren't suggested.
SELECT sqlc.embed(u), false AS is_following
FROM users u
WHERE u.onboarded_at IS NOT NULL
  AND (sqlc.narg('viewer_id')::bigint IS NULL OR u.id <> sqlc.narg('viewer_id'))
  AND NOT EXISTS (SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = sqlc.narg('viewer_id'))
ORDER BY u.followers_count DESC, u.id DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');
