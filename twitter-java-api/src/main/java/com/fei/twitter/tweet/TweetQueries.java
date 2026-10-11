package com.fei.twitter.tweet;

import static com.fei.twitter.shared.web.Sql.instant;
import static com.fei.twitter.shared.web.Sql.nullableLong;
import static com.fei.twitter.shared.web.Sql.ts;

import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.PageRequest;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.Instant;
import java.util.Collection;
import java.util.List;
import java.util.Optional;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * Tweet reads. Every list returns {@link TweetRow}s and pages with keyset cursors: the
 * WHERE clause compares the ORDER BY key of the last row seen instead of using OFFSET.
 * Parameters: {@code :viewer} (nullable) for the like/retweet flags.
 */
@Repository
public class TweetQueries {

    /** The columns of {@link TweetRow}, minus the retweet and sort columns each query adds. */
    static final String SELECT = """
            SELECT t.id, t.user_id, t.content, t.media_type, t.media_url, t.parent_id,
                   t.reply_count, t.like_count, t.retweet_count, t.created_at,
                   EXISTS (SELECT 1 FROM tweet_likes tl WHERE tl.tweet_id = t.id AND tl.user_id = :viewer) AS liked,
                   EXISTS (SELECT 1 FROM retweets rv WHERE rv.tweet_id = t.id AND rv.user_id = :viewer) AS retweeted""";

    /** For lists that aren't timelines: no retweet, no extra sort key. */
    private static final String NO_RETWEET = """
            , NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
              0::float8 AS sort_score, NULL::timestamptz AS sort_time""";

    static final RowMapper<TweetRow> ROW = TweetQueries::map;

    private final JdbcClient jdbc;

    TweetQueries(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    public Optional<TweetRow> byId(long id, Long viewerId) {
        return jdbc.sql(SELECT + NO_RETWEET + " FROM tweets t WHERE t.id = :id")
                .param("id", id)
                .param("viewer", viewerId)
                .query(ROW)
                .optional();
    }

    public List<TweetRow> byIds(Collection<Long> ids, Long viewerId) {
        if (ids.isEmpty()) {
            return List.of();
        }
        return jdbc.sql(SELECT + NO_RETWEET + " FROM tweets t WHERE t.id IN (:ids)")
                .param("ids", ids)
                .param("viewer", viewerId)
                .query(ROW)
                .list();
    }

    /** A tweet's direct replies, oldest first (a thread reads top to bottom). */
    List<TweetRow> replies(long parentId, PageRequest page, Long viewerId) {
        return byCreated("t.parent_id = :owner", parentId, page, viewerId, true);
    }

    List<TweetRow> userReplies(long userId, PageRequest page, Long viewerId) {
        return byCreated("t.user_id = :owner AND t.parent_id IS NOT NULL", userId, page, viewerId, false);
    }

    List<TweetRow> userMedia(long userId, PageRequest page, Long viewerId) {
        return byCreated("t.user_id = :owner AND t.media_url IS NOT NULL", userId, page, viewerId, false);
    }

    /** Lists ordered by (created_at, id); newest first unless {@code ascending}. */
    private List<TweetRow> byCreated(String where, long owner, PageRequest page, Long viewerId, boolean ascending) {
        String op = ascending ? ">" : "<";
        String dir = ascending ? "ASC" : "DESC";
        Cursor after = page.after();
        return jdbc.sql(SELECT + NO_RETWEET + " FROM tweets t WHERE " + where
                        + " AND (CAST(:afterTime AS timestamptz) IS NULL OR (t.created_at, t.id) " + op + " (:afterTime, :afterId))"
                        + " ORDER BY t.created_at " + dir + ", t.id " + dir + " LIMIT :limit")
                .param("owner", owner)
                .param("viewer", viewerId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query(ROW)
                .list();
    }

    /** Tweets a user liked, most recently liked first; sort_time is the like's time. */
    List<TweetRow> userLikes(long userId, PageRequest page, Long viewerId) {
        Cursor after = page.after();
        return jdbc.sql(SELECT + """
                        , NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
                          0::float8 AS sort_score, lk.created_at AS sort_time
                        FROM tweet_likes lk JOIN tweets t ON t.id = lk.tweet_id
                        WHERE lk.user_id = :owner
                          AND (CAST(:afterTime AS timestamptz) IS NULL OR (lk.created_at, lk.tweet_id) < (:afterTime, :afterId))
                        ORDER BY lk.created_at DESC, lk.tweet_id DESC
                        LIMIT :limit""")
                .param("owner", userId)
                .param("viewer", viewerId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query(ROW)
                .list();
    }

    /**
     * Top-level tweets and retweets, newest first, by one user ({@code profile}) or by the
     * accounts {@code :owner} follows. Ordered by (sort_at, kind, sort_id), kind 1 = retweet.
     */
    List<TweetRow> timeline(boolean profile, long owner, PageRequest page, Long viewerId) {
        String tweetsBy = profile ? "t.user_id = :owner" : "t.user_id IN (SELECT following_id FROM follows WHERE follower_id = :owner)";
        String retweetsBy = profile ? "r.user_id = :owner" : "r.user_id IN (SELECT following_id FROM follows WHERE follower_id = :owner)";
        Cursor after = page.after();
        return jdbc.sql("""
                        WITH items AS (
                          SELECT t.id AS tweet_id, NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id,
                                 NULL::timestamptz AS retweeted_at, t.created_at AS sort_at, 0 AS kind, t.id AS sort_id
                          FROM tweets t
                          WHERE %s AND t.parent_id IS NULL
                            AND (CAST(:afterTime AS timestamptz) IS NULL OR t.created_at <= :afterTime)
                          UNION ALL
                          SELECT r.tweet_id, r.id, r.user_id, r.created_at, r.created_at, 1, r.id
                          FROM retweets r
                          WHERE %s
                            AND (CAST(:afterTime AS timestamptz) IS NULL OR r.created_at <= :afterTime)
                        )
                        """.formatted(tweetsBy, retweetsBy)
                        + SELECT + """
                        , i.retweet_id, i.retweeted_by_id, i.retweeted_at, 0::float8 AS sort_score, i.sort_at AS sort_time
                        FROM items i JOIN tweets t ON t.id = i.tweet_id
                        WHERE CAST(:afterTime AS timestamptz) IS NULL
                           OR (i.sort_at, i.kind, i.sort_id) < (:afterTime, :afterKind, :afterId)
                        ORDER BY i.sort_at DESC, i.kind DESC, i.sort_id DESC
                        LIMIT :limit""")
                .param("owner", owner)
                .param("viewer", viewerId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterKind", after == null ? null : after.kind())
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query(ROW)
                .list();
    }

    /**
     * Top-level tweets ranked by tweet_score as of {@code anchor}. Tweets posted after the
     * anchor wait for a refresh, so the ranking doesn't shift while the user scrolls.
     */
    List<TweetRow> forYou(Instant anchor, PageRequest page, Long viewerId) {
        Cursor after = page.after();
        return jdbc.sql(SELECT + """
                        , NULL::bigint AS retweet_id, NULL::bigint AS retweeted_by_id, NULL::timestamptz AS retweeted_at,
                          tweet_score(t.like_count, t.retweet_count, t.reply_count, t.created_at, :anchor) AS sort_score,
                          NULL::timestamptz AS sort_time
                        FROM tweets t
                        WHERE t.parent_id IS NULL
                          AND t.created_at <= :anchor
                          AND (CAST(:afterScore AS float8) IS NULL
                               OR (tweet_score(t.like_count, t.retweet_count, t.reply_count, t.created_at, :anchor), t.id)
                                  < (:afterScore, :afterId))
                        ORDER BY sort_score DESC, t.id DESC
                        LIMIT :limit""")
                .param("anchor", ts(anchor))
                .param("viewer", viewerId)
                .param("afterScore", after == null ? null : after.score())
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query(ROW)
                .list();
    }

    /** Relevance-ranked full-text search; pages by offset (a keyset over ts_rank buys nothing). */
    List<TweetRow> searchFullText(String tsQuery, PageRequest page, Long viewerId) {
        return jdbc.sql(SELECT + NO_RETWEET + """
                         FROM tweets t
                        WHERE t.search_vector @@ to_tsquery('english', :query)
                        ORDER BY ts_rank(t.search_vector, to_tsquery('english', :query)) DESC, t.created_at DESC, t.id DESC
                        LIMIT :limit OFFSET :offset""")
                .param("query", tsQuery)
                .param("viewer", viewerId)
                .param("limit", page.fetchLimit())
                .param("offset", page.offset())
                .query(ROW)
                .list();
    }

    List<TweetRow> searchByHashtag(String tag, PageRequest page, Long viewerId) {
        Cursor after = page.after();
        return jdbc.sql(SELECT + NO_RETWEET + """
                         FROM tweets t
                        JOIN tweet_hashtags th ON th.tweet_id = t.id
                        JOIN hashtags h ON h.id = th.hashtag_id
                        WHERE h.text = lower(:tag)
                          AND (CAST(:afterTime AS timestamptz) IS NULL OR (t.created_at, t.id) < (:afterTime, :afterId))
                        ORDER BY t.created_at DESC, t.id DESC
                        LIMIT :limit""")
                .param("tag", tag)
                .param("viewer", viewerId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query(ROW)
                .list();
    }

    private static TweetRow map(ResultSet rs, int rowNum) throws SQLException {
        return new TweetRow(
                rs.getLong("id"),
                rs.getLong("user_id"),
                rs.getString("content"),
                rs.getString("media_type"),
                rs.getString("media_url"),
                nullableLong(rs, "parent_id"),
                rs.getInt("reply_count"),
                rs.getInt("like_count"),
                rs.getInt("retweet_count"),
                instant(rs, "created_at"),
                rs.getBoolean("liked"),
                rs.getBoolean("retweeted"),
                nullableLong(rs, "retweet_id"),
                nullableLong(rs, "retweeted_by_id"),
                instant(rs, "retweeted_at"),
                rs.getDouble("sort_score"),
                instant(rs, "sort_time"));
    }
}
