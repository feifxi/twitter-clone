package com.fei.twitter.shared.web;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.UUID;
import java.util.regex.Pattern;
import org.slf4j.MDC;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * Gives every request a correlation ID: taken from X-Request-ID when it's a short
 * plain token, generated otherwise. It's echoed in the response, added to every log
 * line (MDC {@code request_id}) and included in error bodies.
 */
@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
public class RequestIdFilter extends OncePerRequestFilter {

    public static final String HEADER = "X-Request-ID";
    public static final String ATTRIBUTE = RequestIdFilter.class.getName();
    private static final Pattern VALID = Pattern.compile("^[A-Za-z0-9_-]{1,64}$");

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        String id = request.getHeader(HEADER);
        if (id == null || !VALID.matcher(id).matches()) {
            id = UUID.randomUUID().toString();
        }
        request.setAttribute(ATTRIBUTE, id);
        response.setHeader(HEADER, id);
        MDC.put("request_id", id);
        try {
            chain.doFilter(request, response);
        } finally {
            MDC.remove("request_id");
        }
    }

    public static String current(HttpServletRequest request) {
        Object id = request.getAttribute(ATTRIBUTE);
        return id instanceof String s ? s : null;
    }

    @Override
    protected boolean shouldNotFilterAsyncDispatch() {
        return false; // SSE and streaming responses log with the same ID
    }
}
