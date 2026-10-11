package com.fei.twitter.security;

import com.fei.twitter.config.AppProperties;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.time.Duration;
import java.time.Instant;
import org.springframework.http.HttpHeaders;
import org.springframework.http.ResponseCookie;
import org.springframework.stereotype.Component;

/**
 * The session lives in two httpOnly cookies, so page scripts (and XSS) can't read the
 * tokens. SameSite=Lax keeps them off cross-site subrequests; cross-origin writes are
 * refused by {@link CrossOriginProtectionFilter}.
 */
@Component
public class SessionCookies {

    public static final String ACCESS = "access_token";
    public static final String REFRESH = "refresh_token";

    private final AppProperties props;

    public SessionCookies(AppProperties props) {
        this.props = props;
    }

    public void set(HttpServletResponse response, String accessToken, Instant accessExpiresAt, String refreshToken, Instant refreshExpiresAt) {
        add(response, ACCESS, accessToken, Duration.between(Instant.now(), accessExpiresAt));
        add(response, REFRESH, refreshToken, Duration.between(Instant.now(), refreshExpiresAt));
    }

    public void clear(HttpServletResponse response) {
        add(response, ACCESS, "", Duration.ZERO);
        add(response, REFRESH, "", Duration.ZERO);
    }

    public static String read(HttpServletRequest request, String name) {
        if (request.getCookies() == null) {
            return null;
        }
        for (Cookie cookie : request.getCookies()) {
            if (name.equals(cookie.getName()) && !cookie.getValue().isEmpty()) {
                return cookie.getValue();
            }
        }
        return null;
    }

    private void add(HttpServletResponse response, String name, String value, Duration maxAge) {
        ResponseCookie.ResponseCookieBuilder cookie = ResponseCookie.from(name, value)
                .path("/")
                .maxAge(maxAge.isNegative() ? Duration.ZERO : maxAge)
                .httpOnly(true)
                .secure(props.secureCookies()) // local dev runs on plain http://localhost
                .sameSite("Lax");
        if (props.cookieDomain() != null && !props.cookieDomain().isBlank()) {
            cookie.domain(props.cookieDomain());
        }
        response.addHeader(HttpHeaders.SET_COOKIE, cookie.build().toString());
    }
}
