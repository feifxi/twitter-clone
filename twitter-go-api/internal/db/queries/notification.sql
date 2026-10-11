-- name: CreateNotification :one
-- Returns no row when deduped by idx_notifications_dedupe.
INSERT INTO notifications (recipient_id, actor_id, tweet_id, type)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE recipient_id = sqlc.arg('recipient_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_limit');

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1 AND is_read = FALSE;
