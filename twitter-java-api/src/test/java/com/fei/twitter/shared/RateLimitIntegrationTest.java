package com.fei.twitter.shared;

import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.FakeGoogle;
import com.fei.twitter.support.TestcontainersConfig;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

@SpringBootTest(properties = {
        "TOKEN_SYMMETRIC_KEY=test-key-0123456789-0123456789-0123456789",
        "GOOGLE_CLIENT_ID=test-client",
        "spring.config.import=",
        "RATE_LIMITS_ENABLED=true",
})
@AutoConfigureMockMvc
@Import({TestcontainersConfig.class, FakeGoogle.class})
class RateLimitIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Test
    void signInEndpointsAllowABurstOfFiveThenAnswer429() {
        int[] statuses = new int[7];
        for (int i = 0; i < statuses.length; i++) {
            statuses[i] = mvc.post().uri("/api/v1/auth/google").contentType(MediaType.APPLICATION_JSON)
                    .content("{\"idToken\":\"bad\"}").exchange().getResponse().getStatus();
        }
        assertThat(statuses).startsWith(401, 401, 401, 401, 401).endsWith(429, 429);
        assertThat(mvc.post().uri("/api/v1/auth/google").contentType(MediaType.APPLICATION_JSON).content("{\"idToken\":\"bad\"}"))
                .hasStatus(429).bodyJson().extractingPath("$.code").isEqualTo("TOO_MANY_REQUESTS");
    }

    @Test
    void limitsArePerRoute() {
        for (int i = 0; i < 6; i++) {
            mvc.post().uri("/api/v1/auth/logout").exchange();
        }
        assertThat(mvc.get().uri("/healthz")).hasStatusOk();
        assertThat(mvc.post().uri("/api/v1/auth/refresh")).as("another route has its own budget").hasStatus(401);
    }
}
