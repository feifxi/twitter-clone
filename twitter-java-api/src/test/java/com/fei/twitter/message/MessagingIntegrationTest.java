package com.fei.twitter.message;

import static com.fei.twitter.support.Api.body;
import static com.fei.twitter.support.Api.collect;
import static com.fei.twitter.support.Api.reply;
import static com.fei.twitter.support.Api.send;
import static com.fei.twitter.support.Api.tweet;
import static com.fei.twitter.support.Sessions.signIn;
import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;

import com.fei.twitter.support.IntegrationTest;
import com.fei.twitter.support.Sessions.TestUser;
import com.jayway.jsonpath.JsonPath;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

@IntegrationTest
class MessagingIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Autowired
    MockMvc rawMvc;

    private org.springframework.test.web.servlet.assertj.MvcTestResult dm(TestUser from, TestUser to, String text) {
        return mvc.post().uri("/api/v1/messages/users/{id}/messages", to.id()).cookie(from.cookie())
                .contentType(MediaType.APPLICATION_JSON).content("{\"content\":\"" + text + "\"}").exchange();
    }

    @Test
    void messagesPageBackwardsInReadingOrder() {
        TestUser alice = signIn(mvc), bob = signIn(mvc);
        long conversation = 0;
        for (int i = 0; i < 5; i++) {
            var result = dm(alice, bob, "m" + i);
            assertThat(result).hasStatus(201);
            conversation = ((Number) JsonPath.read(body(result.getResponse()), "$.conversationId")).longValue();
        }

        String latest = body(mvc.get().uri("/api/v1/messages/conversations/{id}/messages?size=3", conversation).cookie(bob.cookie()).exchange().getResponse());
        assertThat((List<String>) JsonPath.read(latest, "$.items[*].content")).containsExactly("m2", "m3", "m4");
        String older = body(mvc.get().uri("/api/v1/messages/conversations/{id}/messages?size=3&cursor=" + JsonPath.read(latest, "$.nextCursor"), conversation)
                .cookie(bob.cookie()).exchange().getResponse());
        assertThat((List<String>) JsonPath.read(older, "$.items[*].content")).containsExactly("m0", "m1");
        assertThat((Boolean) JsonPath.read(older, "$.hasNext")).isFalse();

        List<Map<String, Object>> inbox = collect(mvc, bob, "/api/v1/messages/conversations", 10);
        assertThat(inbox).hasSize(1);
        assertThat(((Map<?, ?>) inbox.getFirst().get("peer")).get("id")).isEqualTo((int) alice.id());
        assertThat(((Map<?, ?>) inbox.getFirst().get("lastMessage")).get("content")).isEqualTo("m4");

        TestUser outsider = signIn(mvc);
        assertThat(mvc.get().uri("/api/v1/messages/conversations/{id}/messages", conversation).cookie(outsider.cookie())).hasStatus(403);
        assertThat(mvc.post().uri("/api/v1/messages/conversations/{id}/messages", conversation).cookie(outsider.cookie())
                .contentType(MediaType.APPLICATION_JSON).content("{\"content\":\"hi\"}")).hasStatus(403);
    }

    @Test
    void concurrentFirstMessagesShareOneConversation() {
        for (int round = 0; round < 10; round++) {
            TestUser a = signIn(mvc), b = signIn(mvc);
            var first = CompletableFuture.supplyAsync(() -> dm(a, b, "hi"));
            var second = CompletableFuture.supplyAsync(() -> dm(b, a, "hi"));
            Object idA = JsonPath.read(body(first.join().getResponse()), "$.conversationId");
            Object idB = JsonPath.read(body(second.join().getResponse()), "$.conversationId");
            assertThat(idA).isEqualTo(idB);
        }
    }

    @Test
    void lengthIsCountedInCharacters() {
        TestUser a = signIn(mvc), b = signIn(mvc);
        assertThat(dm(a, b, "ก".repeat(2000))).hasStatus(201);
        assertThat(dm(a, b, "ก".repeat(2001))).hasStatus(400);
        assertThat(dm(a, a, "me")).hasStatus(400);
        assertThat(dm(a, b, "  ")).hasStatus(400);
    }

    @Test
    void streamDeliversDirectMessagesAndNotificationsLive() throws Exception {
        TestUser alice = signIn(mvc), bob = signIn(mvc);
        MvcResult stream = rawMvc.perform(get("/api/v1/events").cookie(bob.cookie())).andReturn();
        assertThat(stream.getRequest().isAsyncStarted()).isTrue();

        dm(alice, bob, "live hello");
        long t = tweet(mvc, bob, "like me");
        send(mvc, alice, "POST", "/api/v1/tweets/" + t + "/like");

        String events = stream.getResponse().getContentAsString();
        assertThat(events).contains("event:connected");
        assertThat(events).contains("event:dm").contains("live hello");
        assertThat(events).contains("event:notification").contains("\"type\":\"LIKE\"");
    }

    @Test
    void notificationsListUnreadAndMarkRead() {
        TestUser author = signIn(mvc), fan = signIn(mvc);
        long t = tweet(mvc, author, "original post");
        send(mvc, fan, "POST", "/api/v1/users/" + author.id() + "/follow");
        reply(mvc, fan, t, "nice post");

        assertThat(mvc.get().uri("/api/v1/notifications/unread-count").cookie(author.cookie())).bodyJson().extractingPath("$.count").isEqualTo(2);
        List<Map<String, Object>> items = collect(mvc, author, "/api/v1/notifications", 1);
        assertThat(items).extracting(m -> m.get("type")).containsExactly("REPLY", "FOLLOW");
        Map<String, Object> replyNotification = items.getFirst();
        assertThat(replyNotification.get("tweetContent")).isEqualTo("nice post");
        assertThat(replyNotification.get("originalTweetContent")).isEqualTo("original post");
        assertThat(items.get(1)).doesNotContainKey("originalTweetId");

        assertThat(mvc.post().uri("/api/v1/notifications/mark-read").cookie(author.cookie())).hasStatusOk();
        assertThat(mvc.get().uri("/api/v1/notifications/unread-count").cookie(author.cookie())).bodyJson().extractingPath("$.count").isEqualTo(0);
    }
}
