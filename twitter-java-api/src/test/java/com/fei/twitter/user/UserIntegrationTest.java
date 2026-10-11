package com.fei.twitter.user;

import static com.fei.twitter.support.Sessions.signIn;
import static com.fei.twitter.support.Sessions.signInAs;
import static com.fei.twitter.support.Sessions.unique;
import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.IntegrationTest;
import com.fei.twitter.support.Sessions.TestUser;
import com.jayway.jsonpath.JsonPath;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

@IntegrationTest
class UserIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Autowired
    JdbcClient jdbc;

    @Test
    void profileByIdAndCaseInsensitiveHandle() {
        String name = unique("Alice_");
        TestUser alice = signInAs(mvc, name);

        assertThat(mvc.get().uri("/api/v1/users/{id}", alice.id())).hasStatusOk()
                .bodyJson().extractingPath("$.username").isEqualTo(name);
        assertThat(mvc.get().uri("/api/v1/users/by-username/{u}", name.toLowerCase())).hasStatusOk()
                .bodyJson().extractingPath("$.id").isEqualTo((int) alice.id());
        assertThat(mvc.get().uri("/api/v1/users/by-username/{u}", alice.username()))
                .bodyJson().doesNotHavePath("$.email"); // private fields only in /auth/me
        assertThat(mvc.get().uri("/api/v1/users/999999999")).hasStatus(404).bodyJson().extractingPath("$.code").isEqualTo("NOT_FOUND");
        assertThat(mvc.get().uri("/api/v1/users/abc")).hasStatus(400).bodyJson().extractingPath("$.details[0].field").isEqualTo("id");
    }

    @Test
    void followIsIdempotentUpdatesCountsAndNotifies() {
        TestUser a = signIn(mvc), b = signIn(mvc);

        for (int i = 0; i < 2; i++) {
            assertThat(mvc.post().uri("/api/v1/users/{id}/follow", b.id()).cookie(a.cookie())).hasStatusOk();
        }
        assertThat(mvc.get().uri("/api/v1/users/{id}", b.id()).cookie(a.cookie())).bodyJson()
                .satisfies(json -> {
                    assertThat(json).extractingPath("$.followersCount").isEqualTo(1);
                    assertThat(json).extractingPath("$.isFollowing").isEqualTo(true);
                });
        assertThat(mvc.get().uri("/api/v1/users/{id}", a.id())).bodyJson().extractingPath("$.followingCount").isEqualTo(1);
        assertThat(jdbc.sql("SELECT count(*) FROM notifications WHERE recipient_id = ? AND type = 'FOLLOW'").param(b.id()).query(Long.class).single())
                .as("repeat follows notify once").isEqualTo(1);

        assertThat(mvc.delete().uri("/api/v1/users/{id}/follow", b.id()).cookie(a.cookie())).hasStatusOk();
        assertThat(mvc.delete().uri("/api/v1/users/{id}/follow", b.id()).cookie(a.cookie())).hasStatusOk();
        assertThat(mvc.get().uri("/api/v1/users/{id}", b.id())).bodyJson().extractingPath("$.followersCount").isEqualTo(0);

        assertThat(mvc.post().uri("/api/v1/users/{id}/follow", a.id()).cookie(a.cookie())).hasStatus(400);
        assertThat(mvc.post().uri("/api/v1/users/{id}/follow", b.id())).as("needs a session").hasStatus(401);
    }

    @Test
    void followersPageWithCursors() throws Exception {
        TestUser star = signIn(mvc);
        List<Long> fans = new ArrayList<>();
        for (int i = 0; i < 5; i++) {
            TestUser fan = signIn(mvc);
            mvc.post().uri("/api/v1/users/{id}/follow", star.id()).cookie(fan.cookie()).exchange();
            fans.add(fan.id());
        }

        List<Long> seen = new ArrayList<>();
        String cursor = null;
        do {
            var request = mvc.get().uri("/api/v1/users/{id}/followers?size=2" + (cursor == null ? "" : "&cursor=" + cursor), star.id());
            var response = request.exchange().getResponse();
            String body = response.getContentAsString(StandardCharsets.UTF_8);
            assertThat(response.getStatus()).as(body).isEqualTo(200);
            List<Number> ids = JsonPath.read(body, "$.items[*].id");
            ids.forEach(id -> seen.add(id.longValue()));
            cursor = JsonPath.<Boolean>read(body, "$.hasNext") ? JsonPath.<String>read(body, "$.nextCursor") : null;
        } while (cursor != null);

        assertThat(seen).isEqualTo(fans.reversed()); // most recent follow first, no gaps or repeats
        assertThat(mvc.get().uri("/api/v1/users/{id}/followers?cursor=garbage", star.id()))
                .hasStatus(400).bodyJson().extractingPath("$.details[0].field").isEqualTo("cursor");
    }

    @Test
    void onboardingValidatesHandles() {
        TestUser user = signIn(mvc);
        String taken = signInAs(mvc, unique("taken_")).username();

        assertThat(onboard(user, "explore")).hasStatus(409);
        assertThat(onboard(user, "MESSAGES")).hasStatus(409);
        assertThat(onboard(user, taken.toUpperCase())).hasStatus(409).bodyJson().extractingPath("$.message").isEqualTo("username is already taken");
        assertThat(onboard(user, "a b")).hasStatus(400);

        String mine = unique("mine_");
        assertThat(onboard(user, mine)).hasStatusOk().bodyJson().satisfies(json -> {
            assertThat(json).extractingPath("$.username").isEqualTo(mine);
            assertThat(json).extractingPath("$.onboarded").isEqualTo(true);
        });
    }

    private org.springframework.test.web.servlet.assertj.MvcTestResult onboard(TestUser user, String username) {
        return mvc.post().uri("/api/v1/users/me/onboarding").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON)
                .content("{\"username\":\"" + username + "\",\"displayName\":\"Name\"}").exchange();
    }

    @Test
    void profileUpdateMergesClearsAndValidates() {
        TestUser user = signIn(mvc);
        assertThat(update(user, "{\"bio\":\"hello\",\"location\":\"Bangkok\",\"website\":\"https://me.example\"}")).hasStatusOk();
        assertThat(update(user, "{\"location\":\"\"}")).hasStatusOk().bodyJson().satisfies(json -> {
            assertThat(json).extractingPath("$.bio").isEqualTo("hello"); // omitted: unchanged
            assertThat(json).extractingPath("$.location").isNull(); // "": cleared
        });
        assertThat(update(user, "{\"website\":\"javascript:alert(1)\"}")).hasStatus(400)
                .bodyJson().extractingPath("$.details[0].message").isEqualTo("must be a full http(s) URL");
        assertThat(update(user, "{\"bio\":\"" + "x".repeat(161) + "\"}")).hasStatus(400)
                .bodyJson().extractingPath("$.details[0].message").isEqualTo("must be at most 160 characters");
        assertThat(update(user, "{\"avatarKey\":\"avatars/1/someone-else.png\"}")).hasStatus(400)
                .bodyJson().extractingPath("$.message").isEqualTo("invalid media key");
    }

    private org.springframework.test.web.servlet.assertj.MvcTestResult update(TestUser user, String json) {
        return mvc.put().uri("/api/v1/users/profile").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON).content(json).exchange();
    }

    @Test
    void presignValidatesBeforeTouchingS3() {
        TestUser user = signIn(mvc);
        assertThat(presign(user, "application/pdf", "tweets", 10)).hasStatus(400).bodyJson().extractingPath("$.message").isEqualTo("unsupported content type");
        assertThat(presign(user, "video/mp4", "avatars", 10)).hasStatus(400).bodyJson().extractingPath("$.message").isEqualTo("profile images must be images");
        assertThat(presign(user, "image/png", "avatars", 6 * 1024 * 1024)).hasStatus(400).bodyJson().extractingPath("$.message").isEqualTo("file too large");
        assertThat(presign(user, "image/png", "secrets", 10)).hasStatus(400).bodyJson().extractingPath("$.code").isEqualTo("VALIDATION_ERROR");
    }

    private org.springframework.test.web.servlet.assertj.MvcTestResult presign(TestUser user, String type, String folder, long length) {
        return mvc.post().uri("/api/v1/uploads/presign").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON)
                .content("{\"contentType\":\"" + type + "\",\"folder\":\"" + folder + "\",\"contentLength\":" + length + "}").exchange();
    }
}
