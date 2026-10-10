package com.fei.twitter.message;

import static com.fei.twitter.shared.web.Sql.instant;
import static com.fei.twitter.shared.web.Sql.ts;

import com.fei.twitter.realtime.RealtimeHub;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.user.UserQueries;
import com.fei.twitter.user.UserResponse;
import com.fei.twitter.user.UserView;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;

/** One-to-one direct messages, pushed live to every participant's open streams. */
@Service
public class MessageService {

    public static final String EVENT = "dm";
    private static final int MAX_LENGTH = 2000;

    private record MessageRow(long id, long conversationId, long senderId, String content, Instant createdAt) {}

    private record ConversationRow(long id, Instant updatedAt, UserView peer, long lastId, long lastSenderId, String lastContent, Instant lastAt) {}

    private final JdbcClient jdbc;
    private final UserQueries users;
    private final RealtimeHub hub;
    private final TransactionTemplate tx;

    MessageService(JdbcClient jdbc, UserQueries users, RealtimeHub hub, TransactionTemplate tx) {
        this.jdbc = jdbc;
        this.users = users;
        this.hub = hub;
        this.tx = tx;
    }

    /** The user's conversations, most recently active first. */
    @Transactional(readOnly = true)
    public Page<ConversationResponse> conversations(long userId, PageRequest page) {
        Cursor after = page.after();
        // Conversation columns are aliased: UserQueries.view reads the peer's "id" and "created_at".
        List<ConversationRow> rows = jdbc.sql("SELECT c.id AS conversation_id, c.updated_at AS conversation_updated_at, " + UserQueries.COLUMNS + ", " + UserQueries.FOLLOWING + " AS following,"
                        + """
                         lm.id AS last_id, lm.sender_id AS last_sender_id, lm.content AS last_content, lm.created_at AS last_at
                        FROM conversation_participants self
                        JOIN conversations c ON c.id = self.conversation_id
                        JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> :viewer
                        JOIN users u ON u.id = other.user_id
                        JOIN LATERAL (
                          SELECT dm.id, dm.sender_id, dm.content, dm.created_at FROM direct_messages dm
                          WHERE dm.conversation_id = c.id ORDER BY dm.created_at DESC, dm.id DESC LIMIT 1
                        ) lm ON true
                        WHERE self.user_id = :viewer
                          AND (CAST(:afterTime AS timestamptz) IS NULL OR (lm.created_at, c.id) < (:afterTime, :afterId))
                        ORDER BY lm.created_at DESC, c.id DESC
                        LIMIT :limit""")
                .param("viewer", userId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query((rs, n) -> new ConversationRow(rs.getLong("conversation_id"), instant(rs, "conversation_updated_at"), UserQueries.view(rs, ""),
                        rs.getLong("last_id"), rs.getLong("last_sender_id"), rs.getString("last_content"), instant(rs, "last_at")))
                .list();
        Page<ConversationRow> paged = Page.fromRows(rows, page, r -> Cursor.ofTime(r.lastAt(), r.id()));
        Map<Long, UserView> senders = users.byIds(paged.items().stream().map(ConversationRow::lastSenderId).toList(), userId);
        return paged.map(r -> new ConversationResponse(r.id(), UserResponse.of(r.peer()),
                new MessageResponse(r.lastId(), r.id(), UserResponse.of(senders.get(r.lastSenderId())), r.lastContent(), r.lastAt()),
                r.updatedAt()));
    }

    /** Pages backwards through a conversation: each page holds older messages, oldest-first. */
    @Transactional(readOnly = true)
    public Page<MessageResponse> messages(long userId, long conversationId, PageRequest page) {
        if (!isParticipant(conversationId, userId)) {
            throw ApiException.forbidden("not allowed to access this conversation");
        }
        Cursor after = page.after();
        List<MessageRow> rows = jdbc.sql("""
                        SELECT * FROM direct_messages
                        WHERE conversation_id = :conversation
                          AND (CAST(:afterTime AS timestamptz) IS NULL OR (created_at, id) < (:afterTime, :afterId))
                        ORDER BY created_at DESC, id DESC
                        LIMIT :limit""")
                .param("conversation", conversationId)
                .param("afterTime", after == null ? null : ts(after.time()))
                .param("afterId", after == null ? null : after.id())
                .param("limit", page.fetchLimit())
                .query((rs, n) -> new MessageRow(rs.getLong("id"), rs.getLong("conversation_id"), rs.getLong("sender_id"),
                        rs.getString("content"), instant(rs, "created_at")))
                .list();
        // Rows are newest-first, so the last kept one is the oldest: the next page continues from it.
        Page<MessageRow> paged = Page.fromRows(rows, page, m -> Cursor.ofTime(m.createdAt(), m.id()));
        List<MessageResponse> items = new ArrayList<>(responses(paged.items(), userId));
        return new Page<>(items.reversed(), paged.next());
    }

    /** Sends to the conversation with the recipient, creating it on first contact. */
    public MessageResponse sendToUser(long senderId, long recipientId, String content) {
        if (senderId == recipientId) {
            throw ApiException.badRequest("cannot message yourself");
        }
        String text = normalize(content);
        if (users.byIds(List.of(recipientId), null).isEmpty()) {
            throw ApiException.notFound("user");
        }
        MessageRow created = tx.execute(status -> insert(findOrCreateConversation(senderId, recipientId), senderId, text));
        return delivered(created);
    }

    /** Sends to an existing conversation the sender belongs to. */
    public MessageResponse sendToConversation(long senderId, long conversationId, String content) {
        String text = normalize(content);
        if (!isParticipant(conversationId, senderId)) {
            throw ApiException.forbidden("not allowed to send messages to this conversation");
        }
        return delivered(tx.execute(status -> insert(conversationId, senderId, text)));
    }

    /**
     * The pair's advisory lock (held until commit) serialises "find or create", so two
     * first messages sent at once can't create two conversations.
     */
    private long findOrCreateConversation(long a, long b) {
        jdbc.sql("SELECT pg_advisory_xact_lock(hashtextextended(:key, 0))")
                .param("key", "dm:" + Math.min(a, b) + ":" + Math.max(a, b))
                .query((rs, n) -> null).optional();
        Optional<Long> existing = jdbc.sql("""
                        SELECT c.id FROM conversations c
                        JOIN conversation_participants p1 ON p1.conversation_id = c.id AND p1.user_id = :a
                        JOIN conversation_participants p2 ON p2.conversation_id = c.id AND p2.user_id = :b
                        WHERE (SELECT COUNT(*) FROM conversation_participants cp WHERE cp.conversation_id = c.id) = 2
                        LIMIT 1""")
                .param("a", a).param("b", b).query(Long.class).optional();
        if (existing.isPresent()) {
            return existing.get();
        }
        long id = jdbc.sql("INSERT INTO conversations DEFAULT VALUES RETURNING id").query(Long.class).single();
        for (long user : new long[] {a, b}) {
            jdbc.sql("INSERT INTO conversation_participants (conversation_id, user_id) VALUES (:c, :u)").param("c", id).param("u", user).update();
        }
        return id;
    }

    /** Also bumps the conversation so it sorts first in the inbox. */
    private MessageRow insert(long conversationId, long senderId, String content) {
        MessageRow row = jdbc.sql("INSERT INTO direct_messages (conversation_id, sender_id, content) VALUES (:c, :s, :content) RETURNING *")
                .param("c", conversationId).param("s", senderId).param("content", content)
                .query((rs, n) -> new MessageRow(rs.getLong("id"), rs.getLong("conversation_id"), rs.getLong("sender_id"),
                        rs.getString("content"), instant(rs, "created_at")))
                .single();
        jdbc.sql("UPDATE conversations SET updated_at = now() WHERE id = :c").param("c", conversationId).update();
        return row;
    }

    /** Pushes the stored message to every participant's streams (the sender's other devices too). */
    private MessageResponse delivered(MessageRow row) {
        MessageResponse message = responses(List.of(row), row.senderId()).getFirst();
        jdbc.sql("SELECT user_id FROM conversation_participants WHERE conversation_id = :c").param("c", row.conversationId())
                .query(Long.class).list()
                .forEach(userId -> hub.publish(userId, EVENT, message));
        return message;
    }

    private List<MessageResponse> responses(List<MessageRow> rows, long viewerId) {
        Map<Long, UserView> senders = users.byIds(rows.stream().map(MessageRow::senderId).toList(), viewerId);
        return rows.stream()
                .map(m -> new MessageResponse(m.id(), m.conversationId(), UserResponse.of(senders.get(m.senderId())), m.content(), m.createdAt()))
                .toList();
    }

    private boolean isParticipant(long conversationId, long userId) {
        return jdbc.sql("SELECT EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = :c AND user_id = :u)")
                .param("c", conversationId).param("u", userId).query(Boolean.class).single();
    }

    /** Characters, not bytes: matches the VARCHAR(2000) column. */
    private static String normalize(String content) {
        String text = content == null ? "" : content.strip();
        if (text.isEmpty()) {
            throw ApiException.badRequest("message content is required");
        }
        if (text.codePointCount(0, text.length()) > MAX_LENGTH) {
            throw ApiException.badRequest("message content exceeds 2000 characters");
        }
        return text;
    }
}
