package com.fei.twitter.notification;

import static com.fei.twitter.shared.web.Sql.instant;
import static com.fei.twitter.shared.web.Sql.nullableLong;
import static com.fei.twitter.shared.web.Sql.ts;

import com.fei.twitter.realtime.RealtimeHub;
import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.user.UserQueries;
import com.fei.twitter.user.UserResponse;
import com.fei.twitter.user.UserView;
import java.time.Instant;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.event.TransactionalEventListener;

@Service
public class NotificationService {

    private static final Logger log = LoggerFactory.getLogger(NotificationService.class);
    public static final String EVENT = "notification";

    private record Row(long id, long recipientId, long actorId, Long tweetId, String type, boolean read, Instant createdAt) {}

    private record Preview(String content, String mediaUrl, Long parentId) {}

    private final JdbcClient jdbc;
    private final UserQueries users;
    private final RealtimeHub hub;

    NotificationService(JdbcClient jdbc, UserQueries users, RealtimeHub hub) {
        this.jdbc = jdbc;
        this.users = users;
        this.hub = hub;
    }

    @Transactional(readOnly = true)
    public Page<NotificationResponse> list(long userId, PageRequest page) {
        Cursor after = page.after();
        List<Row> rows = jdbc.sql("""
                        SELECT * FROM notifications
                        WHERE recipient_id = :user
                          AND (CAST(:afterTime AS timestamptz) IS NULL OR (created_at, id) < (:afterTime, :afterId))
                        ORDER BY created_at DESC, id DESC
                        LIMIT :limit""")
                .param("user", userId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query((rs, n) -> new Row(rs.getLong("id"), rs.getLong("recipient_id"), rs.getLong("actor_id"),
                        nullableLong(rs, "tweet_id"), rs.getString("type"), rs.getBoolean("is_read"), instant(rs, "created_at")))
                .list();
        Page<Row> paged = Page.fromRows(rows, page, r -> Cursor.ofTime(r.createdAt(), r.id()));
        return new Page<>(hydrate(paged.items()), paged.next());
    }

    @Transactional(readOnly = true)
    public long unreadCount(long userId) {
        return jdbc.sql("SELECT COUNT(*) FROM notifications WHERE recipient_id = :user AND is_read = FALSE")
                .param("user", userId).query(Long.class).single();
    }

    @Transactional
    public void markAllRead(long userId) {
        jdbc.sql("UPDATE notifications SET is_read = TRUE WHERE recipient_id = :user AND is_read = FALSE").param("user", userId).update();
    }

    /**
     * Pushes a new notification to the recipient's open streams once the creating
     * transaction has committed. Failures are logged: the notification is stored and
     * shows up on the next fetch anyway.
     */
    @TransactionalEventListener
    void onCreated(NotificationCreated event) {
        try {
            jdbc.sql("SELECT * FROM notifications WHERE id = :id").param("id", event.notificationId())
                    .query((rs, n) -> new Row(rs.getLong("id"), rs.getLong("recipient_id"), rs.getLong("actor_id"),
                            nullableLong(rs, "tweet_id"), rs.getString("type"), rs.getBoolean("is_read"), instant(rs, "created_at")))
                    .optional()
                    .ifPresent(row -> hub.publish(row.recipientId(), EVENT, hydrate(List.of(row)).getFirst()));
        } catch (RuntimeException e) {
            log.error("failed to push notification {}", event.notificationId(), e);
        }
    }

    /** Batch-loads actors and tweet previews (plus reply parents) for a page. */
    private List<NotificationResponse> hydrate(List<Row> rows) {
        if (rows.isEmpty()) {
            return List.of();
        }
        Map<Long, UserView> actors = users.byIds(rows.stream().map(Row::actorId).toList(), null);
        Map<Long, Preview> previews = previews(rows.stream().map(Row::tweetId).filter(Objects::nonNull).toList());
        return rows.stream().map(r -> {
            Preview tweet = r.tweetId() == null ? null : previews.get(r.tweetId());
            Preview original = tweet != null && "REPLY".equals(r.type()) && tweet.parentId() != null ? previews.get(tweet.parentId()) : null;
            return new NotificationResponse(r.id(), UserResponse.of(actors.get(r.actorId())), r.tweetId(),
                    tweet == null ? null : tweet.content(), tweet == null ? null : tweet.mediaUrl(),
                    original == null ? null : tweet.parentId(),
                    original == null ? null : original.content(),
                    original == null ? null : original.mediaUrl(),
                    r.type(), r.read(), r.createdAt());
        }).toList();
    }

    /** The notified tweets and, for replies, the tweets they reply to. */
    private Map<Long, Preview> previews(List<Long> tweetIds) {
        Map<Long, Preview> previews = new HashMap<>();
        Set<Long> wanted = new HashSet<>(tweetIds);
        for (int round = 0; round < 2 && !wanted.isEmpty(); round++) {
            jdbc.sql("SELECT id, content, media_url, parent_id FROM tweets WHERE id IN (:ids)").param("ids", wanted)
                    .query((rs, n) -> Map.entry(rs.getLong("id"), new Preview(rs.getString("content"), rs.getString("media_url"), nullableLong(rs, "parent_id"))))
                    .list().forEach(e -> previews.put(e.getKey(), e.getValue()));
            wanted = new HashSet<>();
            for (Preview p : previews.values()) {
                if (p.parentId() != null && !previews.containsKey(p.parentId())) {
                    wanted.add(p.parentId());
                }
            }
        }
        return previews;
    }
}
