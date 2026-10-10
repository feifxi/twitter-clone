package com.fei.twitter.security;

import com.fei.twitter.shared.error.ErrorCode;
import com.fei.twitter.shared.error.ErrorResponse;
import com.fei.twitter.shared.web.RequestIdFilter;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.net.URI;
import java.util.Set;
import java.util.stream.Collectors;
import org.springframework.http.MediaType;
import org.springframework.web.filter.OncePerRequestFilter;
import tools.jackson.databind.json.JsonMapper;

/**
 * Refuses unsafe requests (POST, PUT, DELETE, …) that a browser sends from an origin
 * outside ALLOWED_ORIGINS. Session cookies ride along on such requests, so without this
 * any site could act as the signed-in user (CSRF). Same algorithm as Go's
 * {@code net/http.CrossOriginProtection}: trust Sec-Fetch-Site when present, else
 * compare Origin; requests without either (curl, server-to-server) aren't browser-driven.
 */
public class CrossOriginProtectionFilter extends OncePerRequestFilter {

    private static final Set<String> SAFE_METHODS = Set.of("GET", "HEAD", "OPTIONS");

    private final Set<String> trustedOrigins;
    private final JsonMapper json;

    public CrossOriginProtectionFilter(Iterable<String> trustedOrigins, JsonMapper json) {
        this.trustedOrigins = java.util.stream.StreamSupport.stream(trustedOrigins.spliterator(), false)
                .map(String::trim)
                .collect(Collectors.toUnmodifiableSet());
        this.json = json;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        if (allowed(request)) {
            chain.doFilter(request, response);
            return;
        }
        response.setStatus(ErrorCode.FORBIDDEN.status().value());
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        json.writeValue(response.getOutputStream(),
                ErrorResponse.of(ErrorCode.FORBIDDEN, "cross-origin request refused", RequestIdFilter.current(request)));
    }

    boolean allowed(HttpServletRequest request) {
        if (SAFE_METHODS.contains(request.getMethod())) {
            return true;
        }
        String origin = request.getHeader("Origin");
        String fetchSite = request.getHeader("Sec-Fetch-Site");
        if (fetchSite != null && !fetchSite.isEmpty()) {
            return fetchSite.equals("same-origin") || fetchSite.equals("none") || trusted(origin);
        }
        if (origin == null || origin.isEmpty()) {
            return true;
        }
        return sameHost(origin, request.getHeader("Host")) || trusted(origin);
    }

    private boolean trusted(String origin) {
        return origin != null && trustedOrigins.contains(origin);
    }

    private static boolean sameHost(String origin, String host) {
        try {
            URI uri = URI.create(origin);
            String originHost = uri.getPort() > 0 ? uri.getHost() + ":" + uri.getPort() : uri.getHost();
            return originHost != null && originHost.equalsIgnoreCase(host);
        } catch (IllegalArgumentException e) {
            return false;
        }
    }
}
