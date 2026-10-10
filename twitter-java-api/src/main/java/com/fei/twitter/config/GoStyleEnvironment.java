package com.fei.twitter.config;

import java.net.URI;
import java.util.HashMap;
import java.util.Map;
import org.springframework.boot.EnvironmentPostProcessor;
import org.springframework.boot.SpringApplication;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MapPropertySource;

/**
 * Translates the twitter-go-api environment variables that have no direct Spring
 * equivalent, so both backends run from the same app.env:
 *
 * <ul>
 *   <li>{@code DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=disable} → JDBC URL, user, password
 *   <li>{@code HTTP_SERVER_ADDRESS=0.0.0.0:8080} → server address and port
 *   <li>{@code REDIS_URL} → Spring Data Redis URL
 *   <li>{@code LOG_LEVEL=debug|info|warn|error} → root log level
 *   <li>{@code ENVIRONMENT=production} → JSON (structured) logs
 * </ul>
 *
 * Runs after config data (app.env) has been loaded, so it sees values from either source.
 */
public class GoStyleEnvironment implements EnvironmentPostProcessor, Ordered {

    @Override
    public void postProcessEnvironment(ConfigurableEnvironment env, SpringApplication application) {
        Map<String, Object> props = new HashMap<>();

        String databaseUrl = env.getProperty("DATABASE_URL");
        if (databaseUrl != null && !databaseUrl.isBlank()) {
            props.putAll(jdbcProperties(databaseUrl));
        }

        String address = env.getProperty("HTTP_SERVER_ADDRESS");
        if (address != null && address.contains(":")) {
            int colon = address.lastIndexOf(':');
            props.put("server.address", address.substring(0, colon));
            props.put("server.port", address.substring(colon + 1));
        }

        String redisUrl = env.getProperty("REDIS_URL");
        if (redisUrl != null && !redisUrl.isBlank()) {
            props.put("spring.data.redis.url", redisUrl);
        }

        String logLevel = env.getProperty("LOG_LEVEL");
        if (logLevel != null && !logLevel.isBlank()) {
            props.put("logging.level.root", logLevel.equalsIgnoreCase("warn") ? "WARN" : logLevel.toUpperCase());
        }

        // Production logs are JSON lines (like the Go API's slog JSON handler).
        if ("production".equalsIgnoreCase(env.getProperty("ENVIRONMENT"))) {
            props.put("logging.structured.format.console", "logstash");
        }

        if (!props.isEmpty()) {
            env.getPropertySources().addFirst(new MapPropertySource("goStyleEnvironment", props));
        }
    }

    /** postgresql://user:pass@host:port/db?query → spring.datasource.* (query parameters such as sslmode carry over). */
    static Map<String, Object> jdbcProperties(String databaseUrl) {
        URI uri = URI.create(databaseUrl.replaceFirst("^postgres(ql)?://", "postgresql://"));
        String port = uri.getPort() > 0 ? ":" + uri.getPort() : "";
        String query = uri.getRawQuery() != null ? "?" + uri.getRawQuery() : "";
        Map<String, Object> props = new HashMap<>();
        props.put("spring.datasource.url", "jdbc:postgresql://" + uri.getHost() + port + uri.getRawPath() + query);
        if (uri.getRawUserInfo() != null) {
            String[] parts = uri.getUserInfo().split(":", 2);
            props.put("spring.datasource.username", parts[0]);
            if (parts.length > 1) {
                props.put("spring.datasource.password", parts[1]);
            }
        }
        return props;
    }

    @Override
    public int getOrder() {
        return Ordered.LOWEST_PRECEDENCE;
    }
}
