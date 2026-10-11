package com.fei.twitter.shared.ratelimit;

import com.fei.twitter.config.AppProperties;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.error.ErrorCode;
import com.fei.twitter.shared.ratelimit.RateLimit.Policy;
import io.github.bucket4j.Bandwidth;
import io.github.bucket4j.Bucket;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.RedisScript;
import org.springframework.stereotype.Component;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.HandlerInterceptor;
import org.springframework.web.servlet.HandlerMapping;

/**
 * Enforces {@link RateLimit} policies, keyed by policy, route pattern and client IP.
 * With Redis: a fixed window of burst/rate seconds shared by every instance. Without
 * Redis (or when it fails): an in-memory token bucket per instance.
 */
@Component
public class RateLimitInterceptor implements HandlerInterceptor {

    private static final Logger log = LoggerFactory.getLogger(RateLimitInterceptor.class);
    private static final RedisScript<List> WINDOW = RedisScript.of("""
            local current = redis.call("INCR", KEYS[1])
            if current == 1 then
              redis.call("EXPIRE", KEYS[1], ARGV[1])
            end
            return {current, redis.call("TTL", KEYS[1])}
            """, List.class);

    private final StringRedisTemplate redis; // null when REDIS_URL isn't set
    private final boolean enabled;
    private final Map<String, Bucket> buckets = new ConcurrentHashMap<>();

    public RateLimitInterceptor(ObjectProvider<StringRedisTemplate> redis, org.springframework.core.env.Environment env, AppProperties props) {
        this.redis = env.containsProperty("spring.data.redis.url") ? redis.getIfAvailable() : null;
        this.enabled = props.rateLimitsEnabled();
    }

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) {
        if (!enabled || !request.getRequestURI().startsWith("/api/")) {
            return true;
        }
        Object pattern = request.getAttribute(HandlerMapping.BEST_MATCHING_PATTERN_ATTRIBUTE);
        String route = pattern != null ? pattern.toString() : request.getRequestURI();
        check(Policy.API, route, request, response);
        if (handler instanceof HandlerMethod method && method.hasMethodAnnotation(RateLimit.class)) {
            check(method.getMethodAnnotation(RateLimit.class).value(), route, request, response);
        }
        return true;
    }

    private void check(Policy policy, String route, HttpServletRequest request, HttpServletResponse response) {
        String key = "rl:" + policy.name().toLowerCase() + ":" + route + ":" + request.getRemoteAddr();
        if (!(redis != null ? allowedByRedis(key, policy, response) : allowedLocally(key, policy))) {
            throw new ApiException(ErrorCode.TOO_MANY_REQUESTS, "rate limit exceeded");
        }
    }

    private boolean allowedByRedis(String key, Policy policy, HttpServletResponse response) {
        long window = Math.max(1, (long) Math.ceil(policy.burst / policy.perSecond));
        try {
            List<?> result = redis.execute(WINDOW, List.of(key), Long.toString(window));
            long current = ((Number) result.get(0)).longValue();
            long ttl = ((Number) result.get(1)).longValue();
            response.setHeader("X-RateLimit-Limit", Integer.toString(policy.burst));
            response.setHeader("X-RateLimit-Remaining", Long.toString(Math.max(0, policy.burst - current)));
            if (current > policy.burst) {
                response.setHeader("Retry-After", Long.toString(Math.max(ttl, 1)));
                return false;
            }
            return true;
        } catch (RuntimeException e) {
            log.warn("rate limit: Redis unavailable, using the in-memory limiter", e);
            return allowedLocally(key, policy);
        }
    }

    private boolean allowedLocally(String key, Policy policy) {
        Bucket bucket = buckets.computeIfAbsent(key, k -> Bucket.builder()
                .addLimit(Bandwidth.builder()
                        .capacity(policy.burst)
                        .refillGreedy(Math.max(1, Math.round(policy.perSecond * 10)), Duration.ofSeconds(10))
                        .build())
                .build());
        return bucket.tryConsume(1);
    }
}
