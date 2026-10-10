package com.fei.twitter.search;

import static com.fei.twitter.support.Api.collect;
import static com.fei.twitter.support.Api.id;
import static com.fei.twitter.support.Api.tweet;
import static com.fei.twitter.support.Sessions.signIn;
import static com.fei.twitter.support.Sessions.signInAs;
import static com.fei.twitter.support.Sessions.unique;
import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.IntegrationTest;
import com.fei.twitter.support.Sessions.TestUser;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

@IntegrationTest
class SearchIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    @Test
    void tweetsByFullTextAndHashtag() {
        TestUser user = signIn(mvc);
        String word = unique("zebra");
        String tag = unique("tag");
        long a = tweet(mvc, user, "the " + word + " crossed the road #" + tag);
        long b = tweet(mvc, user, "another " + word + " here");

        assertThat(collect(mvc, null, "/api/v1/search/tweets?q=" + word, 1)).extracting(m -> id(m)).containsExactlyInAnyOrder(a, b);
        assertThat(collect(mvc, null, "/api/v1/search/tweets?q=%23" + tag.toUpperCase(), 10)).extracting(m -> id(m)).containsExactly(a);
        assertThat(collect(mvc, null, "/api/v1/search/tweets?q=" + word + "%20%26%7C!:*", 10)).as("operators are stripped").hasSize(2);
        assertThat(mvc.get().uri("/api/v1/search/tweets?q=")).hasStatus(400);

        assertThat(mvc.get().uri("/api/v1/search/hashtags?q={q}", "#" + tag.substring(0, 6))).hasStatusOk()
                .bodyJson().extractingPath("$[*].text").asArray().contains(tag.toLowerCase());
        assertThat(mvc.get().uri("/api/v1/discovery/trending?limit=500")).hasStatusOk();
    }

    @Test
    void usersByHandleOrNameAndSuggestions() {
        String handle = unique("finder_");
        TestUser found = signInAs(mvc, handle);
        TestUser viewer = signIn(mvc);

        assertThat(collect(mvc, viewer, "/api/v1/search/users?q=" + handle.substring(0, 10), 5)).extracting(m -> id(m)).contains(found.id());

        var suggestions = collect(mvc, viewer, "/api/v1/discovery/users", 50);
        assertThat(suggestions).extracting(m -> id(m)).contains(found.id()).doesNotContain(viewer.id());
        assertThat(suggestions).allMatch(m -> Boolean.FALSE.equals(m.get("isFollowing")));
    }
}
