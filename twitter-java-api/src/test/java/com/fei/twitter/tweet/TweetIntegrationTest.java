package com.fei.twitter.tweet;

import static com.fei.twitter.support.Api.collect;
import static com.fei.twitter.support.Api.id;
import static com.fei.twitter.support.Api.reply;
import static com.fei.twitter.support.Api.send;
import static com.fei.twitter.support.Api.tweet;
import static com.fei.twitter.support.Sessions.signIn;
import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.IntegrationTest;
import com.fei.twitter.support.Sessions.TestUser;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

@IntegrationTest
class TweetIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Autowired
    JdbcClient jdbc;

    @Test
    void followingTimelinePagesTweetsAndRetweetsInOrder() {
        TestUser viewer = signIn(mvc), author = signIn(mvc), retweeter = signIn(mvc);
        send(mvc, viewer, "POST", "/api/v1/users/" + author.id() + "/follow");
        send(mvc, viewer, "POST", "/api/v1/users/" + retweeter.id() + "/follow");

        long[] tweets = new long[5];
        for (int i = 0; i < 5; i++) {
            tweets[i] = tweet(mvc, author, "tweet " + i);
        }
        reply(mvc, author, tweets[0], "a reply never shows on the timeline");
        send(mvc, retweeter, "POST", "/api/v1/tweets/" + tweets[1] + "/retweet");

        List<Map<String, Object>> items = collect(mvc, viewer, "/api/v1/feeds/following", 2);
        assertThat(items).hasSize(6);
        assertThat(id(items.get(0))).isEqualTo(tweets[1]);
        assertThat(((Map<?, ?>) items.get(0).get("retweetedBy")).get("id")).isEqualTo((int) retweeter.id());
        assertThat(items.subList(1, 6)).extracting(m -> id(m)).containsExactly(tweets[4], tweets[3], tweets[2], tweets[1], tweets[0]);
        assertThat(items.subList(1, 6)).allMatch(m -> m.get("retweetedBy") == null);
    }

    @Test
    void forYouPagesWithoutDuplicatesAndRanksByEngagement() {
        TestUser author = signIn(mvc), fan = signIn(mvc);
        long quiet = tweet(mvc, author, "quiet");
        long popular = tweet(mvc, author, "popular");
        send(mvc, fan, "POST", "/api/v1/tweets/" + popular + "/like");
        send(mvc, fan, "POST", "/api/v1/tweets/" + popular + "/retweet");

        List<Map<String, Object>> items = collect(mvc, fan, "/api/v1/feeds/global", 3);
        List<Long> ids = items.stream().map(m -> id(m)).toList();
        assertThat(ids).doesNotHaveDuplicates().contains(quiet, popular);
        assertThat(ids.indexOf(popular)).isLessThan(ids.indexOf(quiet));
        Map<String, Object> mine = items.get(ids.indexOf(popular));
        assertThat(mine.get("isLiked")).isEqualTo(true);
        assertThat(mine.get("isRetweeted")).isEqualTo(true);
    }

    @Test
    void profileTabs() {
        TestUser owner = signIn(mvc), other = signIn(mvc);
        long post = tweet(mvc, owner, "post");
        long replyId = reply(mvc, owner, post, "reply");
        long othersTweet = tweet(mvc, other, "liked");
        send(mvc, owner, "POST", "/api/v1/tweets/" + othersTweet + "/like");

        String base = "/api/v1/users/" + owner.id();
        assertThat(collect(mvc, null, base + "/posts", 10)).extracting(m -> id(m)).containsExactly(post);
        List<Map<String, Object>> replies = collect(mvc, null, base + "/replies", 10);
        assertThat(replies).extracting(m -> id(m)).containsExactly(replyId);
        assertThat(replies.getFirst().get("replyToUsername")).isEqualTo(owner.username());
        assertThat(replies.getFirst().get("replyToTweetId")).isEqualTo((int) post);
        assertThat(collect(mvc, owner, base + "/likes", 10)).extracting(m -> id(m)).containsExactly(othersTweet);

        assertThat(mvc.get().uri(base + "/likes").cookie(other.cookie())).hasStatus(403);
        assertThat(mvc.get().uri(base + "/likes")).as("anonymous").hasStatus(401);
        assertThat(mvc.get().uri("/api/v1/users/999999999/posts")).hasStatus(404);
    }

    @Test
    void likesAndRetweetsAreIdempotentWithCountsAndNotifications() {
        TestUser author = signIn(mvc), fan = signIn(mvc);
        long t = tweet(mvc, author, "hello");

        for (int i = 0; i < 2; i++) {
            send(mvc, fan, "POST", "/api/v1/tweets/" + t + "/like");
            send(mvc, fan, "POST", "/api/v1/tweets/" + t + "/retweet");
        }
        assertThat(mvc.get().uri("/api/v1/tweets/" + t).cookie(fan.cookie())).bodyJson().satisfies(json -> {
            assertThat(json).extractingPath("$.likeCount").isEqualTo(1);
            assertThat(json).extractingPath("$.retweetCount").isEqualTo(1);
            assertThat(json).extractingPath("$.isLiked").isEqualTo(true);
        });
        assertThat(mvc.get().uri("/api/v1/tweets/" + t)).as("anonymous").bodyJson().extractingPath("$.isLiked").isEqualTo(false);
        assertThat(notificationTypes(author)).containsExactlyInAnyOrder("LIKE", "RETWEET");

        for (int i = 0; i < 2; i++) {
            send(mvc, fan, "DELETE", "/api/v1/tweets/" + t + "/like");
            send(mvc, fan, "DELETE", "/api/v1/tweets/" + t + "/retweet");
        }
        assertThat(mvc.get().uri("/api/v1/tweets/" + t)).bodyJson().satisfies(json -> {
            assertThat(json).extractingPath("$.likeCount").isEqualTo(0);
            assertThat(json).extractingPath("$.retweetCount").isEqualTo(0);
        });

        send(mvc, author, "POST", "/api/v1/tweets/" + t + "/like");
        assertThat(notificationTypes(author)).as("liking your own tweet notifies nobody").hasSize(2);
    }

    private List<String> notificationTypes(TestUser recipient) {
        return jdbc.sql("SELECT type FROM notifications WHERE recipient_id = ?").param(recipient.id()).query(String.class).list();
    }

    @Test
    void repliesCountNotifyAndListOldestFirst() {
        TestUser author = signIn(mvc), replier = signIn(mvc);
        long t = tweet(mvc, author, "question");
        long first = reply(mvc, replier, t, "first");
        long second = reply(mvc, replier, t, "second");

        assertThat(mvc.get().uri("/api/v1/tweets/" + t)).bodyJson().extractingPath("$.replyCount").isEqualTo(2);
        assertThat(collect(mvc, null, "/api/v1/tweets/" + t + "/replies", 1)).extracting(m -> id(m)).containsExactly(first, second);
        assertThat(notificationTypes(author)).containsExactly("REPLY", "REPLY");
    }

    @Test
    void deleteRemovesTheThreadAndUnusedHashtags() {
        TestUser author = signIn(mvc), other = signIn(mvc);
        String tag = "tag" + System.nanoTime();
        long t = tweet(mvc, author, "root #" + tag);
        long r = reply(mvc, other, t, "reply #" + tag);

        assertThat(mvc.delete().uri("/api/v1/tweets/" + t).cookie(other.cookie())).hasStatus(403);
        send(mvc, author, "DELETE", "/api/v1/tweets/" + t);

        assertThat(mvc.get().uri("/api/v1/tweets/" + t)).hasStatus(404);
        assertThat(mvc.get().uri("/api/v1/tweets/" + r)).as("replies go with it").hasStatus(404);
        assertThat(jdbc.sql("SELECT count(*) FROM hashtags WHERE text = ?").param(tag).query(Long.class).single()).isZero();
    }

    @Test
    void createValidatesInput() {
        TestUser user = signIn(mvc);
        assertThat(create(user, "{\"content\":\"   \"}")).hasStatus(400).bodyJson().extractingPath("$.message").isEqualTo("tweet must include text or media");
        assertThat(create(user, "{\"content\":\"" + "x".repeat(281) + "\"}")).hasStatus(400)
                .bodyJson().extractingPath("$.details[0].message").isEqualTo("must be at most 280 characters");
        assertThat(create(user, "{\"content\":\"" + "ก".repeat(280) + "\"}")).as("characters, not bytes").hasStatus(201);
        assertThat(create(user, "{\"content\":\"hi\",\"parentId\":999999999}")).hasStatus(400);
        assertThat(create(user, "{\"mediaKey\":\"tweets/1/not-mine.png\"}")).hasStatus(400).bodyJson().extractingPath("$.message").isEqualTo("invalid media key");
        assertThat(create(user, "{\"content\":\"#Java and #java and #spring_boot\"}")).hasStatus(201);
        assertThat(jdbc.sql("SELECT count(*) FROM hashtags WHERE text IN ('java', 'spring_boot')").query(Long.class).single()).isEqualTo(2);
    }

    private org.springframework.test.web.servlet.assertj.MvcTestResult create(TestUser user, String json) {
        return mvc.post().uri("/api/v1/tweets").cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON).content(json).exchange();
    }
}
