package com.fei.twitter.support;

import com.jayway.jsonpath.JsonPath;
import jakarta.servlet.http.Cookie;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.UUID;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockHttpServletResponse;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

/** Signs fresh users in through the real login endpoint (with the fake Google verifier). */
public final class Sessions {

    /** A signed-in test user: send {@link #cookie()} with requests to act as them. */
    public record TestUser(long id, String username, Cookie cookie) {}

    private Sessions() {}

    public static TestUser signIn(MockMvcTester mvc) {
        String email = "u" + UUID.randomUUID().toString().replace("-", "").substring(0, 10) + "@example.com";
        MockHttpServletResponse response = mvc.post().uri("/api/v1/auth/google")
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"idToken\":\"google:" + email + "\"}")
                .exchange()
                .getResponse();
        try {
            String body = response.getContentAsString(StandardCharsets.UTF_8);
            Cookie access = Arrays.stream(response.getCookies()).filter(c -> c.getName().equals("access_token")).findFirst().orElseThrow();
            return new TestUser(((Number) JsonPath.read(body, "$.id")).longValue(), JsonPath.read(body, "$.username"), access);
        } catch (Exception e) {
            throw new IllegalStateException(e);
        }
    }

    /** Signs in and picks a username (onboarding), e.g. to create a recognisable handle. */
    public static TestUser signInAs(MockMvcTester mvc, String username) {
        TestUser user = signIn(mvc);
        mvc.post().uri("/api/v1/users/me/onboarding").cookie(user.cookie())
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"username\":\"" + username + "\",\"displayName\":\"" + username + "\"}")
                .exchange();
        return new TestUser(user.id(), username, user.cookie());
    }

    public static String unique(String prefix) {
        return prefix + UUID.randomUUID().toString().replace("-", "").substring(0, 8);
    }
}
