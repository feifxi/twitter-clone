package com.fei.twitter.user;

import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.List;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/**
 * The follow graph. Follow/unfollow update both users' counters in the same statement
 * as the follow row (no read-modify-write), so concurrent requests can't skew them.
 */
@Repository
public class FollowRepository {

    private final JdbcClient jdbc;

    FollowRepository(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    /** @return false when the follow already existed (counts are only bumped once) */
    boolean follow(long followerId, long targetId) {
        return jdbc.sql("""
                        WITH inserted AS (
                          INSERT INTO follows (follower_id, following_id) VALUES (:follower, :target)
                          ON CONFLICT DO NOTHING
                          RETURNING follower_id
                        ), bump_target AS (
                          UPDATE users SET followers_count = followers_count + 1
                          WHERE id = :target AND EXISTS (SELECT 1 FROM inserted)
                        ), bump_actor AS (
                          UPDATE users SET following_count = following_count + 1
                          WHERE id = :follower AND EXISTS (SELECT 1 FROM inserted)
                        )
                        SELECT EXISTS (SELECT 1 FROM inserted)""")
                .param("follower", followerId)
                .param("target", targetId)
                .query(Boolean.class)
                .single();
    }

    void unfollow(long followerId, long targetId) {
        jdbc.sql("""
                        WITH deleted AS (
                          DELETE FROM follows WHERE follower_id = :follower AND following_id = :target
                          RETURNING follower_id
                        ), bump_target AS (
                          UPDATE users SET followers_count = GREATEST(0, followers_count - 1)
                          WHERE id = :target AND EXISTS (SELECT 1 FROM deleted)
                        ), bump_actor AS (
                          UPDATE users SET following_count = GREATEST(0, following_count - 1)
                          WHERE id = :follower AND EXISTS (SELECT 1 FROM deleted)
                        )
                        SELECT EXISTS (SELECT 1 FROM deleted)""")
                .param("follower", followerId)
                .param("target", targetId)
                .query(Boolean.class)
                .single();
    }

    /** Accounts following userId, most recent follow first. */
    Page<UserView> followers(long userId, PageRequest page, Long viewerId) {
        return list("f.follower_id", "f.following_id", userId, page, viewerId);
    }

    /** Accounts userId follows, most recent follow first. */
    Page<UserView> following(long userId, PageRequest page, Long viewerId) {
        return list("f.following_id", "f.follower_id", userId, page, viewerId);
    }

    private record Row(UserView user, Instant followedAt) {}

    /** Keyset pages ordered by (follow time, other user's id). */
    private Page<UserView> list(String otherColumn, String ownerColumn, long userId, PageRequest page, Long viewerId) {
        Cursor after = page.after();
        List<Row> rows = jdbc.sql("SELECT " + UserQueries.COLUMNS + ", " + UserQueries.FOLLOWING + " AS following, f.created_at AS followed_at"
                        + " FROM follows f JOIN users u ON u.id = " + otherColumn
                        + " WHERE " + ownerColumn + " = :user"
                        + "   AND (CAST(:afterTime AS timestamptz) IS NULL OR (f.created_at, " + otherColumn + ") < (:afterTime, :afterId))"
                        + " ORDER BY f.created_at DESC, " + otherColumn + " DESC"
                        + " LIMIT :limit")
                .param("user", userId)
                .param("viewer", viewerId)
                .param("afterTime", after == null ? null : OffsetDateTime.ofInstant(after.time(), ZoneOffset.UTC))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query((rs, n) -> new Row(UserQueries.view(rs, ""), rs.getObject("followed_at", OffsetDateTime.class).toInstant()))
                .list();
        return Page.fromRows(rows, page, r -> Cursor.ofTime(r.followedAt(), r.user().id())).map(Row::user);
    }
}
