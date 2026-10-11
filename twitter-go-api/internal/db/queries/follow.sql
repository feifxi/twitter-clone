-- name: FollowUser :one
-- Returns false when the follow already existed (counts are only bumped once).
WITH inserted AS (
  INSERT INTO follows (follower_id, following_id) VALUES ($1, $2)
  ON CONFLICT DO NOTHING
  RETURNING follower_id
), bump_target AS (
  UPDATE users SET followers_count = followers_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
), bump_actor AS (
  UPDATE users SET following_count = following_count + 1
  WHERE id = $1 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT EXISTS (SELECT 1 FROM inserted) AS created;

-- name: UnfollowUser :one
WITH deleted AS (
  DELETE FROM follows WHERE follower_id = $1 AND following_id = $2
  RETURNING follower_id
), bump_target AS (
  UPDATE users SET followers_count = GREATEST(0, followers_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
), bump_actor AS (
  UPDATE users SET following_count = GREATEST(0, following_count - 1)
  WHERE id = $1 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT EXISTS (SELECT 1 FROM deleted) AS deleted;
