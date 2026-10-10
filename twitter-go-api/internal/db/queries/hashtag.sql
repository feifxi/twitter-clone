-- name: UpsertHashtag :one
INSERT INTO hashtags (text, usage_count, last_used_at)
VALUES ($1, 1, CURRENT_TIMESTAMP)
ON CONFLICT (text) DO UPDATE
SET usage_count = hashtags.usage_count + 1,
    last_used_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: LinkTweetHashtag :exec
INSERT INTO tweet_hashtags (tweet_id, hashtag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: SearchHashtagsByPrefix :many
SELECT * FROM hashtags
WHERE LOWER(text) LIKE LOWER($1 || '%')
ORDER BY usage_count DESC
LIMIT $2;

-- name: GetTrendingHashtagsLast24h :many
SELECT h.*
FROM hashtags h
JOIN tweet_hashtags th ON th.hashtag_id = h.id
JOIN tweets t ON t.id = th.tweet_id
WHERE t.created_at >= NOW() - INTERVAL '24 hours'
GROUP BY h.id
ORDER BY COUNT(th.tweet_id) DESC, MAX(t.created_at) DESC
LIMIT $1;

-- name: GetTopHashtagsAllTime :many
SELECT * FROM hashtags
ORDER BY usage_count DESC, last_used_at DESC
LIMIT $1;

-- name: ListHashtagUsageToDecrementForDeleteRoot :many
WITH RECURSIVE tweet_tree AS (
  SELECT t.id FROM tweets t WHERE t.id = $1
  UNION ALL
  SELECT t.id FROM tweets t INNER JOIN tweet_tree tt ON t.parent_id = tt.id OR t.retweet_id = tt.id
)
SELECT th.hashtag_id, COUNT(th.hashtag_id)::integer AS decrement_by
FROM tweet_hashtags th
JOIN tweet_tree tt ON th.tweet_id = tt.id
GROUP BY th.hashtag_id;

-- name: DecrementHashtagUsageBy :exec
UPDATE hashtags
SET usage_count = GREATEST(0, usage_count - $2)
WHERE id = $1;

-- name: DeleteUnusedHashtag :exec
DELETE FROM hashtags
WHERE id = $1 AND usage_count <= 0;
