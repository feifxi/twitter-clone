-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expiry_date)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ConsumeRefreshToken :one
-- Deletes and returns the token in one statement, so a refresh token works at
-- most once even when two requests race with it.
DELETE FROM refresh_tokens
WHERE token_hash = $1
RETURNING *;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE token_hash = $1;

-- name: DeleteRefreshTokensByUser :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE user_id = $1 AND expiry_date < now();
