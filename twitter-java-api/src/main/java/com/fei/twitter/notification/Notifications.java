package com.fei.twitter.notification;

import org.springframework.context.ApplicationEventPublisher;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Component;

/**
 * Creates notifications as part of another feature's transaction (like, reply, follow…).
 * Repeats of the same action are deduplicated by idx_notifications_dedupe, and acting
 * on your own content notifies nobody.
 */
@Component
public class Notifications {

    private final JdbcClient jdbc;
    private final ApplicationEventPublisher events;

    Notifications(JdbcClient jdbc, ApplicationEventPublisher events) {
        this.jdbc = jdbc;
        this.events = events;
    }

    /** Must run inside a transaction: the realtime push happens only if it commits. */
    public void create(long recipientId, long actorId, Long tweetId, NotificationType type) {
        if (recipientId == actorId) {
            return;
        }
        jdbc.sql("""
                        INSERT INTO notifications (recipient_id, actor_id, tweet_id, type)
                        VALUES (:recipient, :actor, :tweet, :type)
                        ON CONFLICT DO NOTHING
                        RETURNING id""")
                .param("recipient", recipientId)
                .param("actor", actorId)
                .param("tweet", tweetId)
                .param("type", type.name())
                .query(Long.class)
                .optional()
                .ifPresent(id -> events.publishEvent(new NotificationCreated(id, recipientId)));
    }
}
