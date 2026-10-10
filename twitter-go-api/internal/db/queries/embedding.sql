-- name: ListRelatedTweetsByEmbedding :many
-- Uses cosine distance (<=>) to match the vector_cosine_ops HNSW index.
SELECT tweet_id, content
FROM tweet_embeddings
WHERE embedding <=> sqlc.arg(query_embedding)::vector < sqlc.arg(max_distance)::float8
ORDER BY embedding <=> sqlc.arg(query_embedding)::vector
LIMIT sqlc.arg(result_limit);
