package com.fei.twitter.user;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.Instant;
import java.util.Collection;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * Batch reads of users with the viewer's follow state, used wherever users are
 * embedded in other responses (tweets, notifications, messages). One query per page,
 * never one per item.
 */
@Repository
public class UserQueries {

    /** Columns for {@link #view}; {@code following} must be computed by the query. */
    public static final String COLUMNS = """
            u.id, u.username, u.display_name, u.bio, u.location, u.website, u.avatar_url, u.banner_url,
            u.followers_count, u.following_count, u.created_at""";

    /** "Does :viewer follow u?" (false for anonymous viewers). */
    public static final String FOLLOWING = """
            EXISTS (SELECT 1 FROM follows f WHERE f.follower_id = :viewer AND f.following_id = u.id)""";

    private final JdbcClient jdbc;

    public UserQueries(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    public Map<Long, UserView> byIds(Collection<Long> ids, Long viewerId) {
        if (ids.isEmpty()) {
            return Map.of();
        }
        List<UserView> users = jdbc.sql("SELECT " + COLUMNS + ", " + FOLLOWING + " AS following FROM users u WHERE u.id IN (:ids)")
                .param("ids", ids.stream().distinct().toList())
                .param("viewer", viewerId)
                .query((rs, n) -> view(rs, ""))
                .list();
        return users.stream().collect(Collectors.toMap(UserView::id, Function.identity()));
    }

    /**
     * Maps a row selected with {@link #COLUMNS} (optionally prefixed, e.g. "author_") plus a
     * {@code following} column.
     */
    public static UserView view(ResultSet rs, String prefix) throws SQLException {
        return new UserView(
                rs.getLong(prefix + "id"),
                rs.getString(prefix + "username"),
                rs.getString(prefix + "display_name"),
                rs.getString(prefix + "bio"),
                rs.getString(prefix + "location"),
                rs.getString(prefix + "website"),
                rs.getString(prefix + "avatar_url"),
                rs.getString(prefix + "banner_url"),
                rs.getInt(prefix + "followers_count"),
                rs.getInt(prefix + "following_count"),
                rs.getObject(prefix + "created_at", java.time.OffsetDateTime.class).toInstant(),
                rs.getBoolean(prefix + "following"));
    }

    static Instant instant(ResultSet rs, String column) throws SQLException {
        var value = rs.getObject(column, java.time.OffsetDateTime.class);
        return value == null ? null : value.toInstant();
    }
}
