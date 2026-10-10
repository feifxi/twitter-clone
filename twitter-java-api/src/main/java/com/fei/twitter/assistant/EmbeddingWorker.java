package com.fei.twitter.assistant;

import com.fei.twitter.config.AppProperties;
import com.pgvector.PGvector;
import java.time.Duration;
import java.util.List;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

/**
 * Embeds new tweets for the assistant's retrieval (RAG). Tweets are queued in the
 * embedding_jobs table in the same transaction that creates them; this worker leases due
 * jobs (FOR UPDATE SKIP LOCKED, so several instances can share the queue), retries with
 * exponential backoff, and leaves jobs that keep failing in the table for inspection.
 */
@Component
public class EmbeddingWorker {

    private static final Logger log = LoggerFactory.getLogger(EmbeddingWorker.class);

    static final int BATCH_SIZE = 10;
    static final int MAX_ATTEMPTS = 8;
    static final Duration LEASE = Duration.ofMinutes(5); // a claimed job is due again if not finished by then
    static final Duration BASE_BACKOFF = Duration.ofSeconds(30);
    static final Duration MAX_BACKOFF = Duration.ofHours(1);

    record Job(long tweetId, int attempts, String content) {}

    private final JdbcClient jdbc;
    private final Embedder embedder; // null without Gemini
    private final boolean enabled;

    EmbeddingWorker(JdbcClient jdbc, ObjectProvider<Embedder> embedder, AppProperties props) {
        this.jdbc = jdbc;
        this.embedder = embedder.getIfAvailable();
        this.enabled = props.enableRag() && this.embedder != null;
        if (!enabled) {
            log.warn("RAG disabled: tweets are not embedded (set ENABLE_RAG and GEMINI_API_KEY to enable)");
        }
    }

    /** Drains everything that's due, then waits for the next tick. */
    @Scheduled(fixedDelay = 5_000, initialDelay = 5_000)
    void run() {
        if (!enabled) {
            return;
        }
        try {
            while (processBatch() == BATCH_SIZE) {
                // keep going while full batches come back
            }
        } catch (RuntimeException e) {
            log.error("embedding: claim jobs", e);
        }
    }

    /** Claims up to BATCH_SIZE due jobs and embeds them; returns how many it claimed. */
    int processBatch() {
        List<Job> jobs = jdbc.sql("""
                        WITH due AS (
                          SELECT j.tweet_id FROM embedding_jobs j
                          WHERE j.run_after <= now() AND j.attempts < :maxAttempts
                          ORDER BY j.run_after
                          LIMIT :batch
                          FOR UPDATE SKIP LOCKED
                        )
                        UPDATE embedding_jobs j
                        SET run_after = now() + make_interval(secs => :lease), attempts = j.attempts + 1
                        FROM due, tweets t
                        WHERE j.tweet_id = due.tweet_id AND t.id = j.tweet_id
                        RETURNING j.tweet_id, j.attempts, t.content""")
                .param("maxAttempts", MAX_ATTEMPTS)
                .param("batch", BATCH_SIZE)
                .param("lease", (double) LEASE.toSeconds())
                .query((rs, n) -> new Job(rs.getLong(1), rs.getInt(2), rs.getString(3)))
                .list();
        jobs.forEach(this::process);
        return jobs.size();
    }

    private void process(Job job) {
        if (job.content() == null || job.content().isBlank()) {
            complete(job.tweetId()); // media-only tweets have nothing to embed
            return;
        }
        try {
            float[] vector = embedder.embed(job.content(), Embedder.Task.DOCUMENT);
            jdbc.sql("""
                            INSERT INTO tweet_embeddings (tweet_id, content, embedding) VALUES (:tweet, :content, :embedding)
                            ON CONFLICT (tweet_id) DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding""")
                    .param("tweet", job.tweetId()).param("content", job.content()).param("embedding", new PGvector(vector))
                    .update();
            complete(job.tweetId());
        } catch (RuntimeException e) {
            Duration backoff = backoff(job.attempts());
            log.warn("embedding: job for tweet {} failed (attempt {}), retrying in {}", job.tweetId(), job.attempts(), backoff, e);
            jdbc.sql("UPDATE embedding_jobs SET run_after = now() + make_interval(secs => :backoff), last_error = :error WHERE tweet_id = :tweet")
                    .param("backoff", (double) backoff.toSeconds()).param("error", String.valueOf(e.getMessage())).param("tweet", job.tweetId())
                    .update();
        }
    }

    /** The lease expires and the job runs again if this fails; upserting the embedding is idempotent. */
    private void complete(long tweetId) {
        jdbc.sql("DELETE FROM embedding_jobs WHERE tweet_id = :tweet").param("tweet", tweetId).update();
    }

    /** BASE_BACKOFF doubled per previous attempt, capped at MAX_BACKOFF. */
    static Duration backoff(int attempt) {
        long multiplier = 1L << Math.min(Math.max(attempt - 1, 0), 20);
        Duration d = BASE_BACKOFF.multipliedBy(multiplier);
        return d.compareTo(MAX_BACKOFF) > 0 ? MAX_BACKOFF : d;
    }
}
