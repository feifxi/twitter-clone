package com.fei.twitter.assistant;

import static com.fei.twitter.support.Sessions.signIn;
import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.IntegrationTest;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

/** Without GEMINI_API_KEY the app runs; the assistant answers 500 and tweets aren't queued for embedding. */
@IntegrationTest
class AssistantDisabledIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Test
    void assistantIsUnavailable() {
        var user = signIn(mvc);
        assertThat(mvc.post().uri("/api/v1/assistant").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON).content("{\"query\":\"hi\"}"))
                .hasStatus(500).bodyJson().extractingPath("$.code").isEqualTo("INTERNAL_ERROR");
    }
}
