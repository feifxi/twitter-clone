-- name: ListRelatedTweetsByEmbedding :many
-- Uses cosine distance (<=>) to match the vector_cosine_ops HNSW index.
SELECT tweet_id, content
FROM tweet_embeddings
WHERE embedding <=> sqlc.arg(query_embedding)::vector < sqlc.arg(max_distance)::float8
ORDER BY embedding <=> sqlc.arg(query_embedding)::vector
LIMIT sqlc.arg(result_limit);

-- name: EnqueueEmbeddingJob :exec
INSERT INTO embedding_jobs (tweet_id) VALUES ($1)
ON CONFLICT DO NOTHING;

-- name: ClaimEmbeddingJobs :many
-- Leases due jobs: run_after moves past the lease so a crashed worker's jobs
-- become due again (at-least-once). SKIP LOCKED lets several workers share the queue.
WITH due AS (
  SELECT j.tweet_id
  FROM embedding_jobs j
  WHERE j.run_after <= now() AND j.attempts < sqlc.arg('max_attempts')::int
  ORDER BY j.run_after
  LIMIT sqlc.arg('batch_size')
  FOR UPDATE SKIP LOCKED
)
UPDATE embedding_jobs j
SET run_after = now() + make_interval(secs => sqlc.arg('lease_seconds')::float8),
    attempts = j.attempts + 1
FROM due, tweets t
WHERE j.tweet_id = due.tweet_id AND t.id = j.tweet_id
RETURNING j.tweet_id, j.attempts, t.content;

-- name: CompleteEmbeddingJob :exec
DELETE FROM embedding_jobs WHERE tweet_id = $1;

-- name: RetryEmbeddingJob :exec
UPDATE embedding_jobs
SET run_after = now() + make_interval(secs => sqlc.arg('backoff_seconds')::float8),
    last_error = sqlc.arg('last_error')
WHERE tweet_id = sqlc.arg('tweet_id');

-- name: UpsertTweetEmbedding :exec
INSERT INTO tweet_embeddings (tweet_id, content, embedding)
VALUES ($1, $2, $3)
ON CONFLICT (tweet_id) DO UPDATE
SET content = EXCLUDED.content, embedding = EXCLUDED.embedding;

-- name: CountEmbeddingJobs :one
-- Queue depth for metrics: jobs still to run, and jobs that gave up after max_attempts.
SELECT
  count(*) FILTER (WHERE attempts < sqlc.arg('max_attempts')::int)::bigint  AS pending,
  count(*) FILTER (WHERE attempts >= sqlc.arg('max_attempts')::int)::bigint AS failed
FROM embedding_jobs;
