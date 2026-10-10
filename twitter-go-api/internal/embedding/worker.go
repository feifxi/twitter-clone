package embedding

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/pgvector/pgvector-go"
)

type WorkerConfig struct {
	Interval    time.Duration // how often to poll for due jobs
	BatchSize   int32
	MaxAttempts int32         // jobs that keep failing stay in the table for inspection
	Lease       time.Duration // a claimed job becomes due again if not finished within this
	BaseBackoff time.Duration // first retry delay; doubles per attempt up to MaxBackoff
	MaxBackoff  time.Duration
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Interval:    5 * time.Second,
		BatchSize:   10,
		MaxAttempts: 8,
		Lease:       5 * time.Minute,
		BaseBackoff: 30 * time.Second,
		MaxBackoff:  time.Hour,
	}
}

type Worker struct {
	store    db.Querier
	embedder Embedder
	cfg      WorkerConfig
}

func NewWorker(store db.Querier, embedder Embedder, cfg WorkerConfig) *Worker {
	return &Worker{store: store, embedder: embedder, cfg: cfg}
}

// Run processes due jobs until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		// Drain everything that's due before waiting for the next tick.
		for {
			n, err := w.ProcessBatch(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "embedding: claim jobs", "err", err)
			}
			if err != nil || n < int(w.cfg.BatchSize) {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ProcessBatch claims up to BatchSize due jobs and embeds them, returning how many it claimed.
func (w *Worker) ProcessBatch(ctx context.Context) (int, error) {
	jobs, err := w.store.ClaimEmbeddingJobs(ctx, db.ClaimEmbeddingJobsParams{
		LeaseSeconds: w.cfg.Lease.Seconds(),
		MaxAttempts:  w.cfg.MaxAttempts,
		BatchSize:    w.cfg.BatchSize,
	})
	if err != nil {
		return 0, err
	}
	for _, job := range jobs {
		w.process(ctx, job)
	}
	return len(jobs), nil
}

func (w *Worker) process(ctx context.Context, job db.ClaimEmbeddingJobsRow) {
	if job.Content == nil || *job.Content == "" {
		// Media-only tweets have nothing to embed.
		w.complete(ctx, job.TweetID)
		return
	}

	values, err := w.embedder.Embed(ctx, *job.Content)
	if err == nil {
		vec := pgvector.NewVector(values)
		err = w.store.UpsertTweetEmbedding(ctx, db.UpsertTweetEmbeddingParams{TweetID: job.TweetID, Content: *job.Content, Embedding: &vec})
	}
	if err != nil {
		w.retry(ctx, job, err)
		return
	}
	w.complete(ctx, job.TweetID)
}

func (w *Worker) complete(ctx context.Context, tweetID int64) {
	if err := w.store.CompleteEmbeddingJob(ctx, tweetID); err != nil {
		// The lease will expire and the job will run again; upserting the embedding is idempotent.
		slog.ErrorContext(ctx, "embedding: complete job", "err", err, "tweet_id", tweetID)
	}
}

func (w *Worker) retry(ctx context.Context, job db.ClaimEmbeddingJobsRow, cause error) {
	backoff := w.Backoff(job.Attempts)
	msg := cause.Error()
	slog.WarnContext(ctx, "embedding: job failed, will retry", "err", cause, "tweet_id", job.TweetID, "attempt", job.Attempts, "retry_in", backoff)
	if err := w.store.RetryEmbeddingJob(ctx, db.RetryEmbeddingJobParams{TweetID: job.TweetID, BackoffSeconds: backoff.Seconds(), LastError: &msg}); err != nil {
		slog.ErrorContext(ctx, "embedding: schedule retry", "err", err, "tweet_id", job.TweetID)
	}
}

// Backoff is BaseBackoff doubled per previous attempt, capped at MaxBackoff.
func (w *Worker) Backoff(attempt int32) time.Duration {
	d := float64(w.cfg.BaseBackoff) * math.Pow(2, float64(max(attempt-1, 0)))
	return time.Duration(min(d, float64(w.cfg.MaxBackoff)))
}
