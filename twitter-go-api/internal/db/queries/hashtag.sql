-- name: UpsertHashtag :one
INSERT INTO hashtags (text, usage_count, last_used_at)
VALUES ($1, 1, now())
ON CONFLICT (text) DO UPDATE
SET usage_count = hashtags.usage_count + 1,
    last_used_at = now()
RETURNING *;

-- name: LinkTweetHashtag :exec
INSERT INTO tweet_hashtags (tweet_id, hashtag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: SearchHashtagsByPrefix :many
SELECT * FROM hashtags
WHERE text LIKE lower(sqlc.arg('prefix')) || '%'
ORDER BY usage_count DESC
LIMIT sqlc.arg('result_limit');

-- name: GetTrendingHashtagsLast24h :many
SELECT h.*
FROM hashtags h
JOIN tweet_hashtags th ON th.hashtag_id = h.id
JOIN tweets t ON t.id = th.tweet_id
WHERE t.created_at >= now() - INTERVAL '24 hours'
GROUP BY h.id
ORDER BY COUNT(th.tweet_id) DESC, MAX(t.created_at) DESC
LIMIT $1;

-- name: GetTopHashtagsAllTime :many
SELECT * FROM hashtags
ORDER BY usage_count DESC, last_used_at DESC
LIMIT $1;

-- name: ListHashtagUsageInThread :many
-- Hashtag uses in a tweet and every reply under it. Read before deleting the
-- thread, because ON DELETE CASCADE removes the tweet_hashtags rows.
WITH RECURSIVE thread AS (
  SELECT t.id FROM tweets t WHERE t.id = $1
  UNION ALL
  SELECT t.id FROM tweets t JOIN thread ON t.parent_id = thread.id
)
SELECT th.hashtag_id, COUNT(*)::integer AS uses
FROM tweet_hashtags th
JOIN thread ON th.tweet_id = thread.id
GROUP BY th.hashtag_id;

-- name: DecrementHashtagUsage :exec
UPDATE hashtags
SET usage_count = GREATEST(0, usage_count - sqlc.arg('uses')::integer)
WHERE id = sqlc.arg('id');

-- name: DeleteUnusedHashtag :exec
DELETE FROM hashtags
WHERE id = $1 AND usage_count <= 0;
