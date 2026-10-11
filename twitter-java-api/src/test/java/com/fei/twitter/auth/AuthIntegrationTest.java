package com.fei.twitter.auth;

import static org.assertj.core.api.Assertions.assertThat;

import com.fei.twitter.support.IntegrationTest;
import jakarta.servlet.http.Cookie;
import java.util.Arrays;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockHttpServletResponse;
import org.springframework.test.web.servlet.assertj.MockMvcTester;
import org.springframework.test.web.servlet.assertj.MvcTestResult;

@IntegrationTest
class AuthIntegrationTest {

    @Autowired
    MockMvcTester mvc;

    private MvcTestResult login(String email) {
        return mvc.post().uri("/api/v1/auth/google")
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"idToken\":\"google:" + email + "\"}")
                .exchange();
    }

    private static Map<String, Cookie> cookies(MockHttpServletResponse response) {
        return Arrays.stream(response.getCookies()).collect(Collectors.toMap(Cookie::getName, c -> c, (a, b) -> b));
    }

    private static String email() {
        return "u" + UUID.randomUUID().toString().substring(0, 8) + "@example.com";
    }

    @Test
    void googleLoginCreatesAccountAndSetsHttpOnlyCookies() throws Exception {
        String email = email();
        MvcTestResult result = login(email);

        assertThat(result).hasStatusOk();
        assertThat(result).bodyJson().extractingPath("$.email").isEqualTo(email);
        assertThat(result).bodyJson().extractingPath("$.onboarded").isEqualTo(false);
        assertThat(result).bodyJson().doesNotHavePath("$.accessToken");
        String username = (String) com.jayway.jsonpath.JsonPath.read(result.getResponse().getContentAsString(java.nio.charset.StandardCharsets.UTF_8), "$.username");
        assertThat(username).matches("^[a-z0-9_]{3,11}_[a-z2-7]{8}$");

        var cookies = cookies(result.getResponse());
        assertThat(cookies.get("access_token").isHttpOnly()).isTrue();
        assertThat(cookies.get("refresh_token").isHttpOnly()).isTrue();
        assertThat(result.getResponse().getHeaders("Set-Cookie")).allMatch(h -> h.contains("SameSite=Lax") && h.contains("Path=/"));
        assertThat(cookies.get("access_token").getMaxAge()).isBetween(800, 900);

        // Signing in again finds the same account.
        assertThat(login(email)).bodyJson().extractingPath("$.username").isEqualTo(username);
    }

    @Test
    void unverifiedGoogleEmailIsRejected() {
        assertThat(mvc.post().uri("/api/v1/auth/google").contentType(MediaType.APPLICATION_JSON)
                .content("{\"idToken\":\"unverified:x@example.com\"}"))
                .hasStatus(401).bodyJson().extractingPath("$.code").isEqualTo("UNAUTHORIZED");
    }

    @Test
    void meWorksWithTheCookieOrABearerHeader() {
        var cookies = cookies(login(email()).getResponse());
        Cookie access = cookies.get("access_token");

        assertThat(mvc.get().uri("/api/v1/auth/me").cookie(access)).hasStatusOk();
        assertThat(mvc.get().uri("/api/v1/auth/me").header("Authorization", "Bearer " + access.getValue())).hasStatusOk();
        assertThat(mvc.get().uri("/api/v1/auth/me"))
                .hasStatus(401).bodyJson().extractingPath("$.code").isEqualTo("UNAUTHORIZED");
    }

    @Test
    void refreshRotatesAndOldTokenStopsWorking() {
        Cookie refresh = cookies(login(email()).getResponse()).get("refresh_token");

        MvcTestResult rotated = mvc.post().uri("/api/v1/auth/refresh").cookie(refresh).exchange();
        assertThat(rotated).hasStatusOk();
        Cookie newRefresh = cookies(rotated.getResponse()).get("refresh_token");
        assertThat(newRefresh.getValue()).isNotEqualTo(refresh.getValue());

        MvcTestResult replay = mvc.post().uri("/api/v1/auth/refresh").cookie(refresh).exchange();
        assertThat(replay).hasStatus(401);
        assertThat(replay.getResponse().getHeaders("Set-Cookie")).as("a failed refresh leaves cookies alone").isEmpty();
    }

    @Test
    void staleCredentialsAnswerTokenExpiredEvenOnPublicRoutes() {
        Cookie refresh = cookies(login(email()).getResponse()).get("refresh_token");

        // Only the refresh cookie left (the browser dropped the expired access cookie).
        assertThat(mvc.get().uri("/api/v1/feeds/global").cookie(refresh))
                .hasStatus(401).bodyJson().extractingPath("$.code").isEqualTo("TOKEN_EXPIRED");
        // A garbage access token.
        assertThat(mvc.get().uri("/api/v1/feeds/global").cookie(new Cookie("access_token", "not-a-jwt")))
                .hasStatus(401).bodyJson().extractingPath("$.code").isEqualTo("TOKEN_EXPIRED");
        // The refresh endpoint itself still works with just the refresh cookie.
        assertThat(mvc.post().uri("/api/v1/auth/refresh").cookie(refresh)).hasStatusOk();
    }

    @Test
    void logoutClearsCookiesAndEndsTheSession() {
        Cookie refresh = cookies(login(email()).getResponse()).get("refresh_token");

        MvcTestResult out = mvc.post().uri("/api/v1/auth/logout").cookie(refresh).exchange();
        assertThat(out).hasStatusOk().bodyJson().extractingPath("$.success").isEqualTo(true);
        assertThat(cookies(out.getResponse()).get("refresh_token").getMaxAge()).isZero();
        assertThat(mvc.post().uri("/api/v1/auth/refresh").cookie(refresh)).hasStatus(401);
    }

    @Test
    void crossOriginWritesAreRefused() {
        assertThat(mvc.post().uri("/api/v1/auth/logout").header("Origin", "https://evil.example").header("Sec-Fetch-Site", "cross-site"))
                .hasStatus(403).bodyJson().extractingPath("$.code").isEqualTo("FORBIDDEN");
        assertThat(mvc.post().uri("/api/v1/auth/logout").header("Origin", "http://localhost:3000").header("Sec-Fetch-Site", "same-site"))
                .hasStatusOk();
        assertThat(mvc.post().uri("/api/v1/auth/logout")).as("non-browser client").hasStatusOk();
    }

    @Test
    void errorsHaveTheSharedShapeAndRequestId() {
        MvcTestResult result = mvc.post().uri("/api/v1/auth/google").contentType(MediaType.APPLICATION_JSON).content("{}").exchange();
        assertThat(result).hasStatus(400);
        assertThat(result).bodyJson().extractingPath("$.code").isEqualTo("VALIDATION_ERROR");
        assertThat(result).bodyJson().extractingPath("$.details[0].field").isEqualTo("idToken");
        assertThat(result).bodyJson().extractingPath("$.details[0].message").isEqualTo("this field is required");
        assertThat(result).bodyJson().extractingPath("$.requestId").isEqualTo(result.getResponse().getHeader("X-Request-ID"));

        // Secure by default: unknown routes need a session (401), then they're simply not found.
        assertThat(mvc.get().uri("/api/v1/nope")).hasStatus(401);
        Cookie access = cookies(login(email()).getResponse()).get("access_token");
        assertThat(mvc.get().uri("/api/v1/nope").cookie(access)).hasStatus(404).bodyJson().extractingPath("$.code").isEqualTo("NOT_FOUND");
    }

    @Test
    void healthAndReadiness() {
        assertThat(mvc.get().uri("/healthz")).hasStatusOk().bodyJson().extractingPath("$.status").isEqualTo("ok");
        assertThat(mvc.get().uri("/readyz")).hasStatusOk().bodyJson().extractingPath("$.checks.database").isEqualTo("ok");
    }
}
