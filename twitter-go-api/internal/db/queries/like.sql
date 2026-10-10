-- name: LikeTweet :one
WITH inserted AS (
  INSERT INTO tweet_likes (user_id, tweet_id)
  VALUES ($1, $2)
  ON CONFLICT DO NOTHING
  RETURNING user_id, tweet_id
),
updated AS (
  UPDATE tweets
  SET like_count = like_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT EXISTS(SELECT 1 FROM inserted);

-- name: UnlikeTweet :one
WITH deleted AS (
DELETE FROM tweet_likes
WHERE tweet_likes.user_id = $1 AND tweet_likes.tweet_id = $2
RETURNING user_id, tweet_id
),
updated AS (
  UPDATE tweets
  SET like_count = GREATEST(0, like_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT EXISTS(SELECT 1 FROM deleted);

-- name: IsTweetLiked :one
SELECT EXISTS(
  SELECT 1
  FROM tweet_likes
  WHERE user_id = $1 AND tweet_id = $2
);
