-- name: CreateConversation :one
INSERT INTO conversations DEFAULT VALUES
RETURNING *;

-- name: TouchConversation :exec
UPDATE conversations
SET updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: AddConversationParticipant :exec
INSERT INTO conversation_participants (conversation_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: LockDirectConversationPair :exec
-- Serialises "find or create" for one pair of users until the transaction ends,
-- so two first messages sent at once can't create two conversations.
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(pair_key)::text, 0));

-- name: FindDirectConversation :one
SELECT c.*
FROM conversations c
JOIN conversation_participants p1 ON p1.conversation_id = c.id AND p1.user_id = $1
JOIN conversation_participants p2 ON p2.conversation_id = c.id AND p2.user_id = $2
WHERE (SELECT COUNT(*) FROM conversation_participants cp WHERE cp.conversation_id = c.id) = 2
LIMIT 1;

-- name: IsConversationParticipant :one
SELECT EXISTS(
  SELECT 1
  FROM conversation_participants
  WHERE conversation_id = $1 AND user_id = $2
);

-- name: ListConversationParticipantIDs :many
SELECT user_id
FROM conversation_participants
WHERE conversation_id = $1
ORDER BY user_id ASC;

-- name: ListUserConversations :many
SELECT
  c.id AS conversation_id,
  c.updated_at AS conversation_updated_at,
  sqlc.embed(u),
  lm.id AS last_message_id,
  lm.sender_id AS last_sender_id,
  lm.content AS last_message_content,
  lm.created_at AS last_message_created_at
FROM conversation_participants self
JOIN conversations c ON c.id = self.conversation_id
JOIN conversation_participants otherp ON otherp.conversation_id = c.id AND otherp.user_id <> sqlc.arg('user_id')
JOIN users u ON u.id = otherp.user_id
JOIN LATERAL (
  SELECT dm.id, dm.sender_id, dm.content, dm.created_at
  FROM direct_messages dm
  WHERE dm.conversation_id = c.id
  ORDER BY dm.created_at DESC, dm.id DESC
  LIMIT 1
) lm ON true
WHERE self.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (lm.created_at, c.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY lm.created_at DESC, c.id DESC
LIMIT sqlc.arg('page_limit');

-- name: CreateDirectMessage :one
INSERT INTO direct_messages (conversation_id, sender_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListConversationMessages :many
-- Newest first; each page goes further back in time.
SELECT *
FROM direct_messages
WHERE conversation_id = sqlc.arg('conversation_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_limit');

