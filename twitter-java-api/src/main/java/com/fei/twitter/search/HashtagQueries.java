package com.fei.twitter.search;

import static com.fei.twitter.shared.web.Sql.instant;

import java.util.List;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

@Repository
public class HashtagQueries {

    private static final RowMapper<HashtagResponse> ROW = (rs, n) -> new HashtagResponse(
            rs.getLong("id"), rs.getString("text"), rs.getInt("usage_count"), instant(rs, "last_used_at"), instant(rs, "created_at"));

    private final JdbcClient jdbc;

    HashtagQueries(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    List<HashtagResponse> byPrefix(String prefix, int limit) {
        return jdbc.sql("SELECT * FROM hashtags WHERE text LIKE lower(:prefix) || '%' ORDER BY usage_count DESC LIMIT :limit")
                .param("prefix", prefix).param("limit", limit).query(ROW).list();
    }

    /** Most used in tweets from the last 24 hours. */
    public List<HashtagResponse> trendingLastDay(int limit) {
        return jdbc.sql("""
                        SELECT h.* FROM hashtags h
                        JOIN tweet_hashtags th ON th.hashtag_id = h.id
                        JOIN tweets t ON t.id = th.tweet_id
                        WHERE t.created_at >= now() - INTERVAL '24 hours'
                        GROUP BY h.id
                        ORDER BY COUNT(th.tweet_id) DESC, MAX(t.created_at) DESC
                        LIMIT :limit""")
                .param("limit", limit).query(ROW).list();
    }

    public List<HashtagResponse> topAllTime(int limit) {
        return jdbc.sql("SELECT * FROM hashtags ORDER BY usage_count DESC, last_used_at DESC LIMIT :limit")
                .param("limit", limit).query(ROW).list();
    }
}
