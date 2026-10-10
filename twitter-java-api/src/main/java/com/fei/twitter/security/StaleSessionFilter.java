package com.fei.twitter.security;

import com.fei.twitter.shared.error.ErrorCode;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * The browser drops the access cookie when it expires and keeps only the refresh cookie.
 * Such a request would otherwise look anonymous and get anonymous data on public
 * endpoints (tweets shown as not liked); answer TOKEN_EXPIRED so the client refreshes.
 */
public class StaleSessionFilter extends OncePerRequestFilter {

    private final SessionTokenResolver tokens;
    private final ApiAuthenticationEntryPoint errors;

    public StaleSessionFilter(SessionTokenResolver tokens, ApiAuthenticationEntryPoint errors) {
        this.tokens = tokens;
        this.errors = errors;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        boolean apiRoute = request.getRequestURI().startsWith("/api/")
                && !SessionTokenResolver.SESSION_ENDPOINTS.contains(request.getRequestURI());
        if (apiRoute && tokens.resolve(request) == null && SessionCookies.read(request, SessionCookies.REFRESH) != null) {
            errors.write(request, response, ErrorCode.TOKEN_EXPIRED, "access token expired");
            return;
        }
        chain.doFilter(request, response);
    }
}
