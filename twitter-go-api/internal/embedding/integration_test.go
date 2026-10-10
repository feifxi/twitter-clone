package embedding_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/embedding"
	"github.com/chanombude/twitter-go-api/internal/storage/storagetest"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/stretchr/testify/require"
)

type noopNotifier struct{}

func (noopNotifier) Dispatch(db.Notification) {}

func TestWorker_EmbedsQueuedTweetsAndLeasesJobs(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	author := dbtest.NewUser(t, store)

	// Creating a tweet enqueues its job in the same transaction.
	tweets := tweet.New(tweet.Config{EmbedTweets: true}, store, &storagetest.Fake{}, noopNotifier{})
	tw, err := tweets.Create(ctx, tweet.CreateInput{UserID: author.ID, Content: dbtest.Ptr("embed me")})
	require.NoError(t, err)

	cfg := embedding.DefaultWorkerConfig()
	cfg.BatchSize = 1000 // other tests share the queue

	// A claimed job is leased: a second claim can't take it again.
	leased, err := store.ClaimEmbeddingJobs(ctx, db.ClaimEmbeddingJobsParams{LeaseSeconds: 60, MaxAttempts: cfg.MaxAttempts, BatchSize: cfg.BatchSize})
	require.NoError(t, err)
	require.True(t, containsTweet(leased, tw.ID))
	again, err := store.ClaimEmbeddingJobs(ctx, db.ClaimEmbeddingJobsParams{LeaseSeconds: 60, MaxAttempts: cfg.MaxAttempts, BatchSize: cfg.BatchSize})
	require.NoError(t, err)
	require.False(t, containsTweet(again, tw.ID), "a leased job must not be claimed twice")

	// Once the lease expires, the worker picks it up and stores the embedding.
	require.NoError(t, store.RetryEmbeddingJob(ctx, db.RetryEmbeddingJobParams{TweetID: tw.ID, BackoffSeconds: -1}))
	_, err = embedding.NewWorker(store, fakeEmbedder{}, cfg).ProcessBatch(ctx)
	require.NoError(t, err)

	related, err := store.ListRelatedTweetsByEmbedding(ctx, db.ListRelatedTweetsByEmbeddingParams{QueryEmbedding: vectorPtr(unitVector()), MaxDistance: 0.01, ResultLimit: 1000})
	require.NoError(t, err)
	found := false
	for _, r := range related {
		found = found || r.TweetID == tw.ID
	}
	require.True(t, found, "the worker stored the tweet's embedding")
}

func containsTweet(jobs []db.ClaimEmbeddingJobsRow, tweetID int64) bool {
	for _, j := range jobs {
		if j.TweetID == tweetID {
			return true
		}
	}
	return false
}
