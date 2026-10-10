-- name: CreateTweet :one
INSERT INTO tweets (
  user_id, content, media_type, media_url, parent_id, retweet_id
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetTweet :one
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.id = $1 LIMIT 1;

-- name: DeleteTweetByOwner :one
DELETE FROM tweets
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DecrementParentReplyCount :exec
UPDATE tweets
SET reply_count = GREATEST(0, reply_count - 1)
WHERE id = $1;

-- name: IncrementParentReplyCount :exec
UPDATE tweets
SET reply_count = reply_count + 1
WHERE id = $1;

-- name: CreateRetweet :one
WITH inserted AS (
  INSERT INTO tweets (user_id, retweet_id, media_type)
  VALUES ($1, $2, 'NONE')
  ON CONFLICT DO NOTHING
  RETURNING *
),
updated AS (
  UPDATE tweets
  SET retweet_count = retweet_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT * FROM inserted;

-- name: DeleteRetweetByUser :one
WITH deleted AS (
DELETE FROM tweets
WHERE tweets.user_id = $1 AND tweets.retweet_id = $2
RETURNING *
),
updated AS (
  UPDATE tweets
  SET retweet_count = GREATEST(0, retweet_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT * FROM deleted;

-- name: GetUserRetweet :one
SELECT * FROM tweets
WHERE user_id = $1 AND retweet_id = $2
LIMIT 1;

-- name: ListMediaURLsInThread :many
WITH RECURSIVE tweet_tree AS (
  SELECT t.id, t.media_url
  FROM tweets t
  WHERE t.id = $1
  UNION ALL
  SELECT t.id, t.media_url
  FROM tweets t
  INNER JOIN tweet_tree tt ON t.parent_id = tt.id
)
SELECT media_url
FROM tweet_tree
WHERE media_url IS NOT NULL AND media_url <> '';

-- name: ListForYouFeed :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.parent_id IS NULL
ORDER BY
  (t.like_count * 2 + t.retweet_count * 3 + t.reply_count + 1) /
  POWER((EXTRACT(EPOCH FROM NOW() - t.created_at) / 3600) + 2, 1.8) DESC,
  t.created_at DESC,
  t.id DESC
LIMIT $1 OFFSET $2;

-- name: ListFollowingFeed :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  true AS is_following
FROM tweets t
JOIN follows f ON t.user_id = f.following_id
WHERE f.follower_id = $1
  AND t.parent_id IS NULL
ORDER BY t.created_at DESC
  , t.id DESC
LIMIT $2 OFFSET $3;

-- name: ListUserTweets :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.user_id = $1
  AND t.parent_id IS NULL
ORDER BY t.created_at DESC
  , t.id DESC
LIMIT $2 OFFSET $3;

-- name: ListTweetReplies :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.parent_id = $1
ORDER BY t.created_at ASC
  , t.id ASC
LIMIT $2 OFFSET $3;

-- name: SearchTweetsFullText :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.search_vector @@ to_tsquery('english', $1)
ORDER BY ts_rank(t.search_vector, to_tsquery('english', $1)) DESC, t.created_at DESC, t.id DESC
LIMIT $2 OFFSET $3;

-- name: SearchTweetsByHashtag :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
JOIN tweet_hashtags th ON th.tweet_id = t.id
JOIN hashtags h ON h.id = th.hashtag_id
WHERE LOWER(h.text) = LOWER($1)
ORDER BY t.created_at DESC
  , t.id DESC
LIMIT $2 OFFSET $3;

-- name: GetTweetsByIDs :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM tweets tr WHERE tr.retweet_id = t.id AND tr.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following
FROM tweets t
WHERE t.id = ANY(@tweet_ids::bigint[]);
