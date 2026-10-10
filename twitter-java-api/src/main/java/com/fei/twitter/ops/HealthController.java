package com.fei.twitter.ops;

import com.fasterxml.jackson.annotation.JsonInclude;
import java.util.LinkedHashMap;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.core.env.Environment;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

/** Liveness and readiness probes (same paths and JSON as the Go API). */
@RestController
public class HealthController {

    private static final Logger log = LoggerFactory.getLogger(HealthController.class);

    private final JdbcClient jdbc;
    private final RedisConnectionFactory redis; // null without REDIS_URL

    HealthController(JdbcClient jdbc, ObjectProvider<RedisConnectionFactory> redis, Environment env) {
        this.jdbc = jdbc;
        this.redis = env.containsProperty("spring.data.redis.url") ? redis.getIfAvailable() : null;
    }

    @JsonInclude(JsonInclude.Include.NON_EMPTY)
    record HealthResponse(String status, Map<String, String> checks) {}

    /** The process is up. */
    @GetMapping("/healthz")
    HealthResponse healthz() {
        return new HealthResponse("ok", null);
    }

    /** Dependencies are reachable; failure details are logged, not exposed. */
    @GetMapping("/readyz")
    ResponseEntity<HealthResponse> readyz() {
        Map<String, String> checks = new LinkedHashMap<>();
        checks.put("database", check("database", () -> jdbc.sql("SELECT 1").query(Integer.class).single()));
        if (redis != null) {
            checks.put("redis", check("redis", () -> {
                try (var connection = redis.getConnection()) {
                    connection.ping();
                }
            }));
        }
        boolean ready = !checks.containsValue("error");
        return ResponseEntity.status(ready ? 200 : 503).body(new HealthResponse(ready ? "ready" : "not_ready", checks));
    }

    private static String check(String name, Runnable probe) {
        try {
            probe.run();
            return "ok";
        } catch (RuntimeException e) {
            log.warn("readiness check failed: {}", name, e);
            return "error";
        }
    }
}
