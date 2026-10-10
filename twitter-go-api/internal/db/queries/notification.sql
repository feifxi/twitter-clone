-- name: CreateNotification :one
INSERT INTO notifications (
  recipient_id, actor_id, tweet_id, type
) VALUES (
  $1, $2, $3, $4
)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE recipient_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1 AND is_read = FALSE;
