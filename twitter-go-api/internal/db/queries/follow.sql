-- name: FollowUser :one
WITH inserted AS (
  INSERT INTO follows (follower_id, following_id)
  VALUES ($1, $2)
  ON CONFLICT DO NOTHING
  RETURNING follower_id, following_id
),
update_target AS (
  UPDATE users
  SET followers_count = followers_count + 1
  WHERE id = $2 AND EXISTS (SELECT 1 FROM inserted)
),
update_actor AS (
  UPDATE users
  SET following_count = following_count + 1
  WHERE id = $1 AND EXISTS (SELECT 1 FROM inserted)
)
SELECT EXISTS(SELECT 1 FROM inserted);

-- name: UnfollowUser :one
WITH deleted AS (
  DELETE FROM follows
  WHERE follower_id = $1 AND following_id = $2
  RETURNING follower_id, following_id
),
update_target AS (
  UPDATE users
  SET followers_count = GREATEST(0, followers_count - 1)
  WHERE id = $2 AND EXISTS (SELECT 1 FROM deleted)
),
update_actor AS (
  UPDATE users
  SET following_count = GREATEST(0, following_count - 1)
  WHERE id = $1 AND EXISTS (SELECT 1 FROM deleted)
)
SELECT EXISTS(SELECT 1 FROM deleted);

-- name: GetFollowedUserIDs :many
SELECT following_id FROM follows
WHERE follower_id = $1 AND following_id = ANY($2::int[]);
