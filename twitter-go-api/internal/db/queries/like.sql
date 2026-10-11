-- name: LikeTweet :one
-- Returns false when the like already existed (the count is only bumped once).
WITH inserted AS (
  INSERT INTO tweet_likes (user_id, tweet_id) VALUES ($1, $2)
  ON CONFLICT DO NOTHING
  RETURNING user_id
), bumped AS (
  UPDATE tweets SET like_count = like_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT EXISTS (SELECT 1 FROM inserted) AS created;

-- name: UnlikeTweet :one
WITH deleted AS (
  DELETE FROM tweet_likes tl WHERE tl.user_id = $1 AND tl.tweet_id = $2
  RETURNING tl.user_id
), bumped AS (
  UPDATE tweets SET like_count = GREATEST(0, like_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT EXISTS (SELECT 1 FROM deleted) AS deleted;
