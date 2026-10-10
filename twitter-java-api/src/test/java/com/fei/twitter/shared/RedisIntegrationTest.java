package com.fei.twitter.shared;

import static com.fei.twitter.support.Sessions.signIn;
import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;

import com.fei.twitter.support.FakeGoogle;
import com.fei.twitter.support.Sessions.TestUser;
import com.fei.twitter.support.TestcontainersConfig;
import java.time.Duration;
import org.awaitility.Awaitility;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Import;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.http.MediaType;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.assertj.MockMvcTester;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

/** The Redis code paths: realtime fan-out over pub/sub and the shared rate-limit window. */
@Testcontainers
@SpringBootTest(properties = {
        "TOKEN_SYMMETRIC_KEY=test-key-0123456789-0123456789-0123456789",
        "GOOGLE_CLIENT_ID=test-client",
        "spring.config.import=",
        "RATE_LIMITS_ENABLED=true",
})
@AutoConfigureMockMvc
@Import({TestcontainersConfig.class, FakeGoogle.class})
class RedisIntegrationTest {

    @Container
    static final GenericContainer<?> REDIS = new GenericContainer<>("redis:7.4-alpine").withExposedPorts(6379);

    /**
     * What GoStyleEnvironment derives from REDIS_URL in production (see GoStyleEnvironmentTest);
     * dynamic test properties arrive after that step, so the result is set directly.
     */
    @DynamicPropertySource
    static void redis(DynamicPropertyRegistry registry) {
        registry.add("spring.data.redis.url", () -> "redis://" + REDIS.getHost() + ":" + REDIS.getMappedPort(6379));
    }

    @Autowired
    StringRedisTemplate redis;

    @Autowired
    MockMvcTester mvc;

    @Autowired
    MockMvc rawMvc;

    @Test
    void eventsPublishedOnAnyInstanceReachTheStream() throws Exception {
        TestUser alice = signIn(mvc), bob = signIn(mvc);
        MvcResult stream = rawMvc.perform(get("/api/v1/events").cookie(bob.cookie())).andReturn();

        // As if another API instance published it: only the Redis subscription can deliver this.
        redis.convertAndSend("realtime:events",
                "{\"userId\":" + bob.id() + ",\"type\":\"dm\",\"data\":\"{\\\"content\\\":\\\"from another instance\\\"}\"}");
        Awaitility.await().atMost(Duration.ofSeconds(5))
                .until(() -> stream.getResponse().getContentAsString().contains("from another instance"));

        // And this instance's own sends go out through Redis too (pub/sub is async).
        mvc.post().uri("/api/v1/messages/users/{id}/messages", bob.id()).cookie(alice.cookie())
                .contentType(MediaType.APPLICATION_JSON).content("{\"content\":\"via redis\"}").exchange();
        Awaitility.await().atMost(Duration.ofSeconds(5))
                .until(() -> stream.getResponse().getContentAsString().contains("via redis"));
    }

    @Test
    void rateLimitWindowLivesInRedis() {
        int[] statuses = new int[7];
        for (int i = 0; i < statuses.length; i++) {
            statuses[i] = mvc.post().uri("/api/v1/auth/refresh").exchange().getResponse().getStatus();
        }
        assertThat(statuses).startsWith(401, 401, 401, 401, 401).endsWith(429, 429);
        assertThat(mvc.post().uri("/api/v1/auth/refresh")).hasHeader("Retry-After", "3")
                .satisfies(r -> assertThat(r.getResponse().getHeader("X-RateLimit-Remaining")).isEqualTo("0"));
        assertThat(mvc.get().uri("/readyz")).hasStatusOk().bodyJson().extractingPath("$.checks.redis").isEqualTo("ok");
    }
}
