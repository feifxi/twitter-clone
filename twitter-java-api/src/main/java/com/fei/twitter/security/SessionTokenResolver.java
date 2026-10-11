package com.fei.twitter.security;

import jakarta.servlet.http.HttpServletRequest;
import java.util.Set;
import org.springframework.security.oauth2.server.resource.web.BearerTokenResolver;
import org.springframework.security.oauth2.server.resource.web.DefaultBearerTokenResolver;

/**
 * Finds the access token: an explicit {@code Authorization: Bearer} header (curl, tests)
 * wins over the session cookie (browsers). The sign-in endpoints never authenticate:
 * an expired access cookie must not block refreshing or logging out.
 */
public class SessionTokenResolver implements BearerTokenResolver {

    static final Set<String> SESSION_ENDPOINTS = Set.of("/api/v1/auth/google", "/api/v1/auth/refresh", "/api/v1/auth/logout");

    private final DefaultBearerTokenResolver header = new DefaultBearerTokenResolver();

    @Override
    public String resolve(HttpServletRequest request) {
        if (!request.getRequestURI().startsWith("/api/") || SESSION_ENDPOINTS.contains(request.getRequestURI())) {
            return null;
        }
        String token = header.resolve(request);
        return token != null ? token : SessionCookies.read(request, SessionCookies.ACCESS);
    }
}
