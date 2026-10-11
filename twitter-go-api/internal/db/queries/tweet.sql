-- Every tweet-listing query returns the same columns: the tweet, the viewer's
-- flags, and (for timelines) who retweeted it. That shared shape lets the tweet
-- package hydrate any of them the same way. Cursors are keyset: they compare the
-- ORDER BY key of the last row seen instead of using OFFSET.

-- name: CreateTweet :one
INSERT INTO tweets (user_id, content, media_type, media_url, parent_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTweet :one
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.id = sqlc.arg('id');

-- name: GetTweetsByIDs :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.id = ANY(sqlc.arg('tweet_ids')::bigint[]);

-- name: DeleteTweetByOwner :one
DELETE FROM tweets
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: IncrementReplyCount :exec
UPDATE tweets SET reply_count = reply_count + 1 WHERE id = $1;

-- name: DecrementReplyCount :exec
UPDATE tweets SET reply_count = GREATEST(0, reply_count - 1) WHERE id = $1;

-- name: ListMediaURLsInThread :many
-- Media of a tweet and every reply under it (all removed by ON DELETE CASCADE).
WITH RECURSIVE thread AS (
  SELECT t.id, t.media_url FROM tweets t WHERE t.id = $1
  UNION ALL
  SELECT t.id, t.media_url FROM tweets t JOIN thread ON t.parent_id = thread.id
)
SELECT media_url FROM thread WHERE media_url IS NOT NULL;

-- name: CreateRetweet :one
-- Idempotent: returns false when the user had already retweeted.
WITH inserted AS (
  INSERT INTO retweets (user_id, tweet_id) VALUES ($1, $2)
  ON CONFLICT (user_id, tweet_id) DO NOTHING
  RETURNING id
), bumped AS (
  UPDATE tweets SET retweet_count = retweet_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT EXISTS (SELECT 1 FROM inserted) AS created;

-- name: DeleteRetweet :one
WITH deleted AS (
  DELETE FROM retweets r WHERE r.user_id = $1 AND r.tweet_id = $2
  RETURNING r.id
), bumped AS (
  UPDATE tweets SET retweet_count = GREATEST(0, retweet_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT EXISTS (SELECT 1 FROM deleted) AS deleted;

-- name: ListForYouFeed :many
-- Top-level tweets ranked by tweet_score as of @anchor. Tweets posted after the
-- anchor wait for a refresh; counts can still change between pages, so an item
-- may occasionally shift across a page boundary.
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
  tweet_score(t.like_count, t.retweet_count, t.reply_count, t.created_at, sqlc.arg('anchor')) AS score
FROM tweets t
WHERE t.parent_id IS NULL
  AND t.created_at <= sqlc.arg('anchor')
  AND (sqlc.narg('cursor_score')::float8 IS NULL
       OR (tweet_score(t.like_count, t.retweet_count, t.reply_count, t.created_at, sqlc.arg('anchor')), t.id)
          < (sqlc.narg('cursor_score')::float8, sqlc.narg('cursor_id')::bigint))
ORDER BY score DESC, t.id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListFollowingTimeline :many
-- Top-level tweets and retweets by accounts the viewer follows, newest first.
-- Items are ordered by (sort_at, kind, sort_id): kind 0 = tweet, 1 = retweet.
WITH items AS (
  SELECT t.id AS tweet_id, NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
         t.created_at AS sort_at, 0 AS kind, t.id AS sort_id
  FROM tweets t
  JOIN follows f ON f.following_id = t.user_id
  WHERE f.follower_id = sqlc.arg('viewer_id')
    AND t.parent_id IS NULL
    AND (sqlc.narg('cursor_time')::timestamptz IS NULL OR t.created_at <= sqlc.narg('cursor_time'))
  UNION ALL
  SELECT r.tweet_id, r.id, r.user_id, r.created_at, r.created_at, 1, r.id
  FROM retweets r
  JOIN follows f ON f.following_id = r.user_id
  WHERE f.follower_id = sqlc.arg('viewer_id')
    AND (sqlc.narg('cursor_time')::timestamptz IS NULL OR r.created_at <= sqlc.narg('cursor_time'))
)
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.arg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.arg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.arg('viewer_id')) AS is_following,
  i.retweet_id, i.retweeted_by_id, i.retweeted_at
FROM items i
JOIN tweets t ON t.id = i.tweet_id
WHERE sqlc.narg('cursor_time')::timestamptz IS NULL
   OR (i.sort_at, i.kind, i.sort_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_kind')::int, sqlc.narg('cursor_id')::bigint)
ORDER BY i.sort_at DESC, i.kind DESC, i.sort_id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListProfileTimeline :many
-- A user's top-level tweets and retweets, newest first. Same ordering as ListFollowingTimeline.
WITH items AS (
  SELECT t.id AS tweet_id, NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
         t.created_at AS sort_at, 0 AS kind, t.id AS sort_id
  FROM tweets t
  WHERE t.user_id = sqlc.arg('user_id')
    AND t.parent_id IS NULL
    AND (sqlc.narg('cursor_time')::timestamptz IS NULL OR t.created_at <= sqlc.narg('cursor_time'))
  UNION ALL
  SELECT r.tweet_id, r.id, r.user_id, r.created_at, r.created_at, 1, r.id
  FROM retweets r
  WHERE r.user_id = sqlc.arg('user_id')
    AND (sqlc.narg('cursor_time')::timestamptz IS NULL OR r.created_at <= sqlc.narg('cursor_time'))
)
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  i.retweet_id, i.retweeted_by_id, i.retweeted_at
FROM items i
JOIN tweets t ON t.id = i.tweet_id
WHERE sqlc.narg('cursor_time')::timestamptz IS NULL
   OR (i.sort_at, i.kind, i.sort_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_kind')::int, sqlc.narg('cursor_id')::bigint)
ORDER BY i.sort_at DESC, i.kind DESC, i.sort_id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListUserReplies :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.user_id = sqlc.arg('user_id')
  AND t.parent_id IS NOT NULL
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (t.created_at, t.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListUserMedia :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.user_id = sqlc.arg('user_id')
  AND t.media_url IS NOT NULL
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (t.created_at, t.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListUserLikes :many
-- Tweets a user liked, most recently liked first; the cursor is the like's time.
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl2 WHERE tl2.tweet_id = t.id AND tl2.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
  tl.created_at AS liked_at
FROM tweet_likes tl
JOIN tweets t ON t.id = tl.tweet_id
WHERE tl.user_id = sqlc.arg('user_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (tl.created_at, tl.tweet_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY tl.created_at DESC, tl.tweet_id DESC
LIMIT sqlc.arg('page_limit');

-- name: ListTweetReplies :many
-- Oldest first, so a thread reads top to bottom.
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.parent_id = sqlc.arg('parent_id')
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (t.created_at, t.id) > (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY t.created_at ASC, t.id ASC
LIMIT sqlc.arg('page_limit');

-- name: SearchTweetsByHashtag :many
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
JOIN tweet_hashtags th ON th.tweet_id = t.id
JOIN hashtags h ON h.id = th.hashtag_id
WHERE h.text = lower(sqlc.arg('tag'))
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (t.created_at, t.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint))
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg('page_limit');

-- name: SearchTweetsFullText :many
-- Relevance-ranked, so it pages by offset: a keyset cursor over ts_rank buys nothing here.
SELECT sqlc.embed(t),
  EXISTS(SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = sqlc.narg('viewer_id')) AS is_liked,
  EXISTS(SELECT 1 FROM retweets r WHERE r.tweet_id = t.id AND r.user_id = sqlc.narg('viewer_id')) AS is_retweeted,
  EXISTS(SELECT 1 FROM follows f WHERE f.following_id = t.user_id AND f.follower_id = sqlc.narg('viewer_id')) AS is_following,
  NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at
FROM tweets t
WHERE t.search_vector @@ to_tsquery('english', sqlc.arg('query'))
ORDER BY ts_rank(t.search_vector, to_tsquery('english', sqlc.arg('query'))) DESC, t.created_at DESC, t.id DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');
