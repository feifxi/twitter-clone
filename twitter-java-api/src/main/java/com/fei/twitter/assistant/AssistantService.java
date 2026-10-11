package com.fei.twitter.assistant;

import com.fei.twitter.config.AppProperties;
import com.pgvector.PGvector;
import java.util.ArrayList;
import java.util.List;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.chat.client.ChatClient;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.messages.Message;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.ai.chat.model.ChatModel;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Flux;

/** Streams Gemini answers grounded in tweets similar to the question (RAG). */
@Service
public class AssistantService {

    private static final Logger log = LoggerFactory.getLogger(AssistantService.class);
    private static final int MAX_HISTORY = 30;
    private static final int RAG_RESULTS = 5;
    private static final double RAG_MAX_COSINE_DISTANCE = 0.5;
    private static final String PREAMBLE = """
            You are the helpful, professional, and polite AI assistant for "Chanom Twitter", a social media platform and Twitter clone.
            Your goal is to provide accurate, concise, and friendly information to users of Chanom Twitter.
            You have access to the user's recent [Chat History] and [Timeline Context] from their feed.
            Prioritize the Timeline Context when answering questions about the app's content or the user's recent experience.
            Maintain conversational continuity and a helpful tone consistent with a premium social experience.
            CRITICAL: Your memory is limited to the provided history. If asked about older context you don't have, politely state you don't have access to that part of the conversation and ask for clarification.""";

    public record Turn(boolean fromModel, String text) {}

    private final ChatClient chat; // null without GEMINI_API_KEY
    private final Embedder embedder;
    private final JdbcClient jdbc;
    private final boolean rag;

    AssistantService(ObjectProvider<ChatModel> chatModel, ObjectProvider<Embedder> embedder, JdbcClient jdbc, AppProperties props) {
        ChatModel model = chatModel.getIfAvailable(); // absent without GEMINI_API_KEY
        this.chat = model == null ? null : ChatClient.create(model);
        this.embedder = embedder.getIfAvailable();
        this.jdbc = jdbc;
        this.rag = props.enableRag() && this.embedder != null;
    }

    /** The reply as text chunks. Failing before the stream starts is an exception; mid-stream, an error signal. */
    public Flux<String> chat(String query, List<Turn> history) {
        if (chat == null) {
            throw new IllegalStateException("assistant is not configured (GEMINI_API_KEY is not set)");
        }
        List<Turn> recent = history.size() > MAX_HISTORY ? history.subList(history.size() - MAX_HISTORY, history.size()) : history;
        List<Message> messages = new ArrayList<>();
        for (Turn turn : recent) {
            messages.add(turn.fromModel() ? new AssistantMessage(turn.text()) : new UserMessage(turn.text()));
        }
        return chat.prompt()
                .system(PREAMBLE + "\n\n[Timeline Context]\n" + timelineContext(query))
                .messages(messages)
                .user(query)
                .stream()
                .content()
                .filter(text -> !text.isEmpty());
    }

    /** Never fails the chat: retrieval problems degrade to an answer without context. */
    private String timelineContext(String query) {
        if (!rag) {
            return "Timeline context is currently disabled.";
        }
        float[] vector;
        try {
            vector = embedder.embed(query, Embedder.Task.QUERY);
        } catch (RuntimeException e) {
            log.error("failed to vectorize query for RAG, falling back to base chat", e);
            return "Timeline context unavailable (vectorization failed).";
        }
        try {
            // Cosine distance (<=>) matches the vector_cosine_ops HNSW index.
            List<String> tweets = jdbc.sql("""
                            SELECT content FROM tweet_embeddings
                            WHERE embedding <=> CAST(:q AS vector) < :maxDistance
                            ORDER BY embedding <=> CAST(:q AS vector)
                            LIMIT :limit""")
                    .param("q", new PGvector(vector)).param("maxDistance", RAG_MAX_COSINE_DISTANCE).param("limit", RAG_RESULTS)
                    .query(String.class).list();
            return tweets.isEmpty() ? "No recent matching posts found in the timeline." : "- " + String.join("\n- ", tweets);
        } catch (RuntimeException e) {
            log.error("failed to retrieve RAG context", e);
            return "Timeline context unavailable (DB error).";
        }
    }
}
