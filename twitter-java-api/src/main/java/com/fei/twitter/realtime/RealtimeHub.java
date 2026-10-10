package com.fei.twitter.realtime;

import jakarta.annotation.PreDestroy;
import java.io.IOException;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.core.env.Environment;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.listener.ChannelTopic;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;
import tools.jackson.databind.json.JsonMapper;

/**
 * Fans events out to a user's open SSE streams. With Redis, events published on any
 * instance reach subscribers on every instance; without it, delivery is in-process
 * (fine for a single instance).
 */
@Component
public class RealtimeHub {

    private static final Logger log = LoggerFactory.getLogger(RealtimeHub.class);
    private static final String CHANNEL = "realtime:events";

    /** What travels over Redis: the recipient, the SSE event name and its JSON data. */
    record Envelope(long userId, String type, String data) {}

    private final Map<Long, Set<SseEmitter>> streams = new ConcurrentHashMap<>();
    private final JsonMapper json;
    private final StringRedisTemplate redis;
    private final RedisMessageListenerContainer listener;

    RealtimeHub(JsonMapper json, ObjectProvider<StringRedisTemplate> redis, ObjectProvider<RedisConnectionFactory> connections, Environment env) {
        this.json = json;
        boolean useRedis = env.containsProperty("spring.data.redis.url");
        this.redis = useRedis ? redis.getIfAvailable() : null;
        if (this.redis != null) {
            listener = new RedisMessageListenerContainer();
            listener.setConnectionFactory(connections.getObject());
            listener.addMessageListener((message, pattern) -> {
                try {
                    deliver(json.readValue(message.getBody(), Envelope.class));
                } catch (RuntimeException e) {
                    log.error("realtime: malformed event from Redis", e);
                }
            }, new ChannelTopic(CHANNEL));
            listener.afterPropertiesSet();
            listener.start();
        } else {
            listener = null;
        }
    }

    /** Sends {@code payload} (as JSON) to every stream the user has open, on any instance. */
    public void publish(long userId, String type, Object payload) {
        Envelope envelope = new Envelope(userId, type, json.writeValueAsString(payload));
        if (redis == null) {
            deliver(envelope);
            return;
        }
        try {
            redis.convertAndSend(CHANNEL, json.writeValueAsString(envelope));
        } catch (RuntimeException e) {
            log.error("realtime: Redis publish failed, delivering locally", e);
            deliver(envelope); // users connected to this instance still get it
        }
    }

    /** Registers a stream; it unregisters itself when it completes, times out or fails. */
    public SseEmitter subscribe(long userId) {
        SseEmitter emitter = new SseEmitter(0L); // no timeout: the client keeps it open
        streams.computeIfAbsent(userId, id -> ConcurrentHashMap.newKeySet()).add(emitter);
        Runnable remove = () -> streams.computeIfPresent(userId, (id, set) -> {
            set.remove(emitter);
            return set.isEmpty() ? null : set;
        });
        emitter.onCompletion(remove);
        emitter.onTimeout(remove);
        emitter.onError(e -> remove.run());
        try {
            emitter.send(SseEmitter.event().reconnectTime(3000).name("connected").data("{\"status\":\"ok\"}"));
        } catch (IOException e) {
            emitter.completeWithError(e);
        }
        return emitter;
    }

    private void deliver(Envelope envelope) {
        for (SseEmitter emitter : streams.getOrDefault(envelope.userId(), Set.of())) {
            try {
                emitter.send(SseEmitter.event().name(envelope.type()).data(envelope.data()));
            } catch (IOException | IllegalStateException e) {
                emitter.completeWithError(e); // a gone client; onError unregisters it
            }
        }
    }

    /** An SSE comment keeps proxies from closing idle connections. */
    @Scheduled(fixedRate = 15_000)
    void heartbeat() {
        streams.values().forEach(set -> set.forEach(emitter -> {
            try {
                emitter.send(SseEmitter.event().comment("ping"));
            } catch (IOException | IllegalStateException e) {
                emitter.completeWithError(e);
            }
        }));
    }

    /** On shutdown, end open streams instead of letting graceful shutdown wait them out. */
    @PreDestroy
    void close() {
        streams.values().forEach(set -> set.forEach(SseEmitter::complete));
        if (listener != null) {
            listener.stop();
        }
    }

    int openStreams(long userId) {
        return streams.getOrDefault(userId, Set.of()).size();
    }
}
