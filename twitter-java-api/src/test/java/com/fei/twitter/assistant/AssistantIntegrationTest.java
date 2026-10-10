package com.fei.twitter.assistant;

import static com.fei.twitter.support.Api.tweet;
import static com.fei.twitter.support.Sessions.signIn;
import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.asyncDispatch;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;

import com.fei.twitter.support.FakeGoogle;
import com.fei.twitter.support.Sessions.TestUser;
import com.fei.twitter.support.TestcontainersConfig;
import java.time.Duration;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.model.ChatModel;
import org.springframework.ai.chat.model.ChatResponse;
import org.springframework.ai.chat.model.Generation;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Import;
import org.springframework.context.annotation.Primary;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.assertj.MockMvcTester;
import reactor.core.publisher.Flux;

/** The assistant and the RAG pipeline with fake Gemini models (no network). */
@SpringBootTest(properties = {
        "TOKEN_SYMMETRIC_KEY=test-key-0123456789-0123456789-0123456789",
        "GOOGLE_CLIENT_ID=test-client",
        "spring.config.import=",
        "RATE_LIMITS_ENABLED=false",
        "GEMINI_API_KEY=test-key",
})
@AutoConfigureMockMvc
@Import({TestcontainersConfig.class, FakeGoogle.class, AssistantIntegrationTest.FakeGemini.class})
class AssistantIntegrationTest {

    /** Everything the fake chat model was asked (to check the RAG context reached it). */
    static final AtomicReference<Prompt> LAST_PROMPT = new AtomicReference<>();

    @TestConfiguration(proxyBeanMethods = false)
    static class FakeGemini {

        @Bean
        @Primary
        ChatModel fakeChatModel() {
            return new ChatModel() {
                @Override
                public ChatResponse call(Prompt prompt) {
                    throw new UnsupportedOperationException();
                }

                @Override
                public Flux<ChatResponse> stream(Prompt prompt) {
                    LAST_PROMPT.set(prompt);
                    return Flux.just("Hello", " there").map(t -> new ChatResponse(List.of(new Generation(new AssistantMessage(t)))));
                }
            };
        }

        /** Every text maps to the same unit vector, so every stored tweet matches every question. */
        @Bean
        @Primary
        Embedder fakeEmbedder() {
            return (text, task) -> {
                if (text.contains("FAIL")) {
                    throw new IllegalStateException("quota exceeded");
                }
                float[] v = new float[Embedder.DIMENSIONS];
                v[0] = 1;
                return v;
            };
        }
    }

    @Autowired
    MockMvcTester mvc;

    @Autowired
    MockMvc rawMvc;

    @Autowired
    EmbeddingWorker worker;

    @Autowired
    JdbcClient jdbc;

    @Test
    void newTweetsAreEmbeddedAndUsedAsContext() throws Exception {
        TestUser user = signIn(mvc);
        String fact = "the cafeteria serves pad thai on fridays " + System.nanoTime();
        long id = tweet(mvc, user, fact);
        assertThat(count("SELECT count(*) FROM embedding_jobs WHERE tweet_id = ?", id)).as("queued with the tweet").isEqualTo(1);

        while (worker.processBatch() > 0) {
            // drain
        }
        assertThat(count("SELECT count(*) FROM tweet_embeddings WHERE tweet_id = ?", id)).isEqualTo(1);
        assertThat(count("SELECT count(*) FROM embedding_jobs WHERE tweet_id = ?", id)).as("job done").isZero();

        MvcResult started = rawMvc.perform(post("/api/v1/assistant").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON)
                .content("{\"query\":\"what's for lunch on friday?\",\"history\":[{\"role\":\"user\",\"text\":\"hi\"},{\"role\":\"model\",\"text\":\"hello\"}]}"))
                .andReturn();
        started.getAsyncResult(Duration.ofSeconds(5).toMillis());
        String body = rawMvc.perform(asyncDispatch(started)).andReturn().getResponse().getContentAsString();

        assertThat(body).contains("data:{\"text\":\"Hello\"}").contains("data:{\"text\":\" there\"}").contains("event:done");
        String system = LAST_PROMPT.get().getSystemMessage().getText();
        assertThat(system).contains("[Timeline Context]").contains(fact);
        assertThat(LAST_PROMPT.get().getInstructions()).hasSize(4); // system, 2 history turns, the question
    }

    @Test
    void failedEmbeddingsAreRetriedWithBackoff() {
        TestUser user = signIn(mvc);
        long id = tweet(mvc, user, "FAIL to embed " + System.nanoTime());
        while (worker.processBatch() > 0) {
            // drain: the failing job is rescheduled into the future, so it isn't claimed again
        }
        assertThat(jdbc.sql("SELECT attempts FROM embedding_jobs WHERE tweet_id = ?").param(id).query(Integer.class).single()).isEqualTo(1);
        assertThat(jdbc.sql("SELECT last_error FROM embedding_jobs WHERE tweet_id = ?").param(id).query(String.class).single()).isEqualTo("quota exceeded");
        assertThat(jdbc.sql("SELECT run_after > now() + interval '20 seconds' FROM embedding_jobs WHERE tweet_id = ?").param(id).query(Boolean.class).single()).isTrue();
    }

    @Test
    void backoffDoublesAndIsCapped() {
        assertThat(EmbeddingWorker.backoff(1)).isEqualTo(Duration.ofSeconds(30));
        assertThat(EmbeddingWorker.backoff(3)).isEqualTo(Duration.ofMinutes(2));
        assertThat(EmbeddingWorker.backoff(20)).isEqualTo(Duration.ofHours(1));
    }

    @Test
    void validatesTheRequest() {
        TestUser user = signIn(mvc);
        assertThat(mvc.post().uri("/api/v1/assistant").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON)
                .content("{\"query\":\"hi\",\"history\":[{\"role\":\"system\",\"text\":\"x\"}]}"))
                .hasStatus(400).bodyJson().extractingPath("$.details[0].field").isEqualTo("history[0].role");
        assertThat(mvc.post().uri("/api/v1/assistant").contentType(MediaType.APPLICATION_JSON).content("{\"query\":\"hi\"}")).hasStatus(401);
    }

    private long count(String sql, long id) {
        return jdbc.sql(sql).param(id).query(Long.class).single();
    }
}
