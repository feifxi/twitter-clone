package com.fei.twitter.tweet;

import com.fei.twitter.config.AppProperties;
import com.fei.twitter.media.MediaKeys;
import com.fei.twitter.media.MediaStorage;
import com.fei.twitter.notification.NotificationType;
import com.fei.twitter.notification.Notifications;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;

/** Posting, deleting, liking and retweeting. Counters change in the same SQL statement as their rows. */
@Service
public class TweetService {

    private static final Pattern HASHTAG = Pattern.compile("(?i)(?:^|\\s)#([a-z0-9_]+)");

    private final JdbcClient jdbc;
    private final TweetQueries queries;
    private final TweetHydrator hydrator;
    private final Notifications notifications;
    private final MediaStorage storage;
    private final TransactionTemplate tx;
    private final boolean embedTweets;

    TweetService(JdbcClient jdbc, TweetQueries queries, TweetHydrator hydrator, Notifications notifications,
                 MediaStorage storage, TransactionTemplate tx, AppProperties props) {
        this.jdbc = jdbc;
        this.queries = queries;
        this.hydrator = hydrator;
        this.notifications = notifications;
        this.storage = storage;
        this.tx = tx;
        // RAG needs Gemini to embed tweets; the embedding worker drains the job queue.
        this.embedTweets = props.enableRag() && props.geminiConfigured();
    }

    public TweetView get(long tweetId, Long viewerId) {
        return hydrator.hydrate(List.of(row(tweetId, viewerId)), viewerId).getFirst();
    }

    /** A tweet's direct replies, oldest first. */
    public Page<TweetView> replies(long tweetId, PageRequest page, Long viewerId) {
        row(tweetId, viewerId);
        return hydrator.page(queries.replies(tweetId, page, viewerId), page, viewerId, TweetRow::createdCursor);
    }

    public TweetView create(long userId, String content, Long parentId, String mediaKey) {
        String text = content == null || content.isBlank() ? null : content.strip();
        String mediaType = null;
        String mediaUrl = null;
        if (mediaKey != null && !mediaKey.isEmpty()) {
            MediaKeys.validateOwned(mediaKey, MediaKeys.TWEETS, userId);
            mediaType = MediaKeys.typeOf(mediaKey);
            mediaUrl = storage.publicUrl(mediaKey);
        }
        if (text == null && mediaUrl == null) {
            throw ApiException.badRequest("tweet must include text or media");
        }

        String type = mediaType;
        String url = mediaUrl;
        long id;
        try {
            id = tx.execute(status -> insert(userId, text, parentId, type, url));
        } catch (RuntimeException e) {
            storage.delete(mediaKey); // the upload belongs to nothing now
            throw e;
        }
        return get(id, userId);
    }

    private long insert(long userId, String text, Long parentId, String mediaType, String mediaUrl) {
        long id = jdbc.sql("""
                        INSERT INTO tweets (user_id, content, media_type, media_url, parent_id)
                        VALUES (:user, :content, :mediaType, :mediaUrl, :parent) RETURNING id""")
                .param("user", userId).param("content", text).param("mediaType", mediaType)
                .param("mediaUrl", mediaUrl).param("parent", parentId)
                .query(Long.class).single();

        if (parentId != null) {
            long parentAuthor = jdbc.sql("UPDATE tweets SET reply_count = reply_count + 1 WHERE id = :id RETURNING user_id")
                    .param("id", parentId).query(Long.class).optional()
                    .orElseThrow(() -> ApiException.notFound("parent tweet"));
            notifications.create(parentAuthor, userId, id, NotificationType.REPLY);
        }
        if (text != null && embedTweets) {
            // Same transaction as the tweet: the job exists if and only if the tweet does.
            jdbc.sql("INSERT INTO embedding_jobs (tweet_id) VALUES (:id) ON CONFLICT DO NOTHING").param("id", id).update();
        }
        if (text != null) {
            for (String tag : hashtags(text)) {
                long tagId = jdbc.sql("""
                                INSERT INTO hashtags (text, usage_count, last_used_at) VALUES (:text, 1, now())
                                ON CONFLICT (text) DO UPDATE SET usage_count = hashtags.usage_count + 1, last_used_at = now()
                                RETURNING id""")
                        .param("text", tag).query(Long.class).single();
                jdbc.sql("INSERT INTO tweet_hashtags (tweet_id, hashtag_id) VALUES (:tweet, :tag) ON CONFLICT DO NOTHING")
                        .param("tweet", id).param("tag", tagId).update();
            }
        }
        return id;
    }

    /** Deletes the tweet and its whole reply thread (ON DELETE CASCADE), then their hosted media. */
    public void delete(long userId, long tweetId) {
        TweetRow tweet = row(tweetId, null);
        if (tweet.userId() != userId) {
            throw ApiException.forbidden("you can only delete your own tweets");
        }
        List<String> mediaUrls = jdbc.sql("""
                        WITH RECURSIVE thread AS (
                          SELECT t.id, t.media_url FROM tweets t WHERE t.id = :id
                          UNION ALL
                          SELECT t.id, t.media_url FROM tweets t JOIN thread ON t.parent_id = thread.id
                        )
                        SELECT media_url FROM thread WHERE media_url IS NOT NULL""")
                .param("id", tweetId).query(String.class).list();

        tx.executeWithoutResult(status -> {
            // Read hashtag usage for the whole thread first: the cascade removes the replies' links too.
            List<long[]> usage = jdbc.sql("""
                            WITH RECURSIVE thread AS (
                              SELECT t.id FROM tweets t WHERE t.id = :id
                              UNION ALL
                              SELECT t.id FROM tweets t JOIN thread ON t.parent_id = thread.id
                            )
                            SELECT th.hashtag_id, COUNT(*) AS uses FROM tweet_hashtags th JOIN thread ON th.tweet_id = thread.id
                            GROUP BY th.hashtag_id""")
                    .param("id", tweetId).query((rs, n) -> new long[] {rs.getLong(1), rs.getLong(2)}).list();
            jdbc.sql("DELETE FROM tweets WHERE id = :id AND user_id = :user").param("id", tweetId).param("user", userId).update();
            if (tweet.parentId() != null) {
                jdbc.sql("UPDATE tweets SET reply_count = GREATEST(0, reply_count - 1) WHERE id = :id").param("id", tweet.parentId()).update();
            }
            for (long[] u : usage) {
                jdbc.sql("UPDATE hashtags SET usage_count = GREATEST(0, usage_count - :uses) WHERE id = :id")
                        .param("uses", u[1]).param("id", u[0]).update();
                jdbc.sql("DELETE FROM hashtags WHERE id = :id AND usage_count <= 0").param("id", u[0]).update();
            }
        });

        // After the commit, so a failed delete never loses media. External URLs are skipped.
        new LinkedHashSet<>(mediaUrls).forEach(url -> storage.keyFromPublicUrl(url).ifPresent(storage::delete));
    }

    @Transactional
    public void like(long userId, long tweetId) {
        TweetRow tweet = row(tweetId, null);
        boolean created = jdbc.sql("""
                        WITH inserted AS (
                          INSERT INTO tweet_likes (user_id, tweet_id) VALUES (:user, :tweet) ON CONFLICT DO NOTHING RETURNING user_id
                        ), bumped AS (
                          UPDATE tweets SET like_count = like_count + 1 WHERE id = :tweet AND EXISTS (SELECT 1 FROM inserted)
                        )
                        SELECT EXISTS (SELECT 1 FROM inserted)""")
                .param("user", userId).param("tweet", tweetId).query(Boolean.class).single();
        if (created) {
            notifications.create(tweet.userId(), userId, tweetId, NotificationType.LIKE);
        }
    }

    @Transactional
    public void unlike(long userId, long tweetId) {
        row(tweetId, null);
        jdbc.sql("""
                        WITH deleted AS (
                          DELETE FROM tweet_likes tl WHERE tl.user_id = :user AND tl.tweet_id = :tweet RETURNING tl.user_id
                        ), bumped AS (
                          UPDATE tweets SET like_count = GREATEST(0, like_count - 1) WHERE id = :tweet AND EXISTS (SELECT 1 FROM deleted)
                        )
                        SELECT EXISTS (SELECT 1 FROM deleted)""")
                .param("user", userId).param("tweet", tweetId).query(Boolean.class).single();
    }

    /** Idempotent; returns the tweet as the retweeter now sees it. */
    public TweetView retweet(long userId, long tweetId) {
        TweetRow tweet = row(tweetId, null);
        tx.executeWithoutResult(status -> {
            boolean created = jdbc.sql("""
                            WITH inserted AS (
                              INSERT INTO retweets (user_id, tweet_id) VALUES (:user, :tweet)
                              ON CONFLICT (user_id, tweet_id) DO NOTHING RETURNING id
                            ), bumped AS (
                              UPDATE tweets SET retweet_count = retweet_count + 1 WHERE id = :tweet AND EXISTS (SELECT 1 FROM inserted)
                            )
                            SELECT EXISTS (SELECT 1 FROM inserted)""")
                    .param("user", userId).param("tweet", tweetId).query(Boolean.class).single();
            if (created) {
                notifications.create(tweet.userId(), userId, tweetId, NotificationType.RETWEET);
            }
        });
        return get(tweetId, userId);
    }

    /** Idempotent: undoing a retweet that doesn't exist succeeds. */
    @Transactional
    public void undoRetweet(long userId, long tweetId) {
        row(tweetId, null);
        jdbc.sql("""
                        WITH deleted AS (
                          DELETE FROM retweets r WHERE r.user_id = :user AND r.tweet_id = :tweet RETURNING r.id
                        ), bumped AS (
                          UPDATE tweets SET retweet_count = GREATEST(0, retweet_count - 1) WHERE id = :tweet AND EXISTS (SELECT 1 FROM deleted)
                        )
                        SELECT EXISTS (SELECT 1 FROM deleted)""")
                .param("user", userId).param("tweet", tweetId).query(Boolean.class).single();
    }

    private TweetRow row(long tweetId, Long viewerId) {
        return queries.byId(tweetId, viewerId).orElseThrow(() -> ApiException.notFound("tweet"));
    }

    /** Lowercased, unique, in order of appearance. */
    static Set<String> hashtags(String content) {
        Set<String> tags = new LinkedHashSet<>();
        Matcher m = HASHTAG.matcher(content);
        while (m.find()) {
            tags.add(m.group(1).toLowerCase(Locale.ROOT));
        }
        return tags;
    }
}
