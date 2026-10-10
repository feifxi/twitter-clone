package embedding_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/embedding"
	"github.com/stretchr/testify/require"
)

type fakeEmbedder struct{ err error }

func (f fakeEmbedder) Embed(context.Context, string) ([]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return unitVector(), nil
}

// unitVector is non-zero: cosine distance to an all-zero vector is undefined (NaN).
func unitVector() []float32 {
	v := make([]float32, embedding.Dimensions)
	v[0] = 1
	return v
}

func TestBackoffDoublesAndCaps(t *testing.T) {
	w := embedding.NewWorker(nil, nil, embedding.WorkerConfig{BaseBackoff: 30 * time.Second, MaxBackoff: 5 * time.Minute})
	for attempt, want := range map[int32]time.Duration{1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute, 4: 4 * time.Minute, 5: 5 * time.Minute, 9: 5 * time.Minute} {
		require.Equal(t, want, w.Backoff(attempt), "attempt %d", attempt)
	}
}

func TestProcessBatch(t *testing.T) {
	ctx := context.Background()
	text := "hello"
	jobs := []db.ClaimEmbeddingJobsRow{
		{TweetID: 1, Attempts: 1, Content: &text},
		{TweetID: 2, Attempts: 3, Content: nil}, // media-only: nothing to embed
	}

	t.Run("success_stores_embedding_and_completes", func(t *testing.T) {
		var stored, completed []int64
		store := &dbtest.MockStore{
			ClaimEmbeddingJobsFn: func(context.Context, db.ClaimEmbeddingJobsParams) ([]db.ClaimEmbeddingJobsRow, error) {
				return jobs, nil
			},
			UpsertTweetEmbeddingFn: func(_ context.Context, arg db.UpsertTweetEmbeddingParams) error {
				stored = append(stored, arg.TweetID)
				return nil
			},
			CompleteEmbeddingJobFn: func(_ context.Context, id int64) error {
				completed = append(completed, id)
				return nil
			},
		}

		n, err := embedding.NewWorker(store, fakeEmbedder{}, embedding.DefaultWorkerConfig()).ProcessBatch(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.Equal(t, []int64{1}, stored)
		require.Equal(t, []int64{1, 2}, completed)
	})

	t.Run("failure_schedules_a_retry_with_backoff", func(t *testing.T) {
		var retried db.RetryEmbeddingJobParams
		store := &dbtest.MockStore{
			ClaimEmbeddingJobsFn: func(context.Context, db.ClaimEmbeddingJobsParams) ([]db.ClaimEmbeddingJobsRow, error) {
				return jobs[:1], nil
			},
			RetryEmbeddingJobFn: func(_ context.Context, arg db.RetryEmbeddingJobParams) error {
				retried = arg
				return nil
			},
			CompleteEmbeddingJobFn: func(context.Context, int64) error {
				t.Fatal("a failed job must not be completed")
				return nil
			},
		}

		_, err := embedding.NewWorker(store, fakeEmbedder{err: errors.New("quota exceeded")}, embedding.DefaultWorkerConfig()).ProcessBatch(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(1), retried.TweetID)
		require.Equal(t, 30.0, retried.BackoffSeconds)
		require.Equal(t, "quota exceeded", *retried.LastError)
	})
}
