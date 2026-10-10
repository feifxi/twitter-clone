package com.fei.twitter.security;

import com.fei.twitter.shared.error.ErrorCode;
import com.fei.twitter.shared.error.ErrorResponse;
import com.fei.twitter.shared.web.RequestIdFilter;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.springframework.http.MediaType;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.oauth2.server.resource.InvalidBearerTokenException;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.stereotype.Component;
import tools.jackson.databind.json.JsonMapper;

/**
 * 401 responses distinguish two cases (see docs/api-contract.md):
 * TOKEN_EXPIRED when stale credentials were sent (the client should refresh),
 * UNAUTHORIZED when none were (the user is signed out).
 */
@Component
public class ApiAuthenticationEntryPoint implements AuthenticationEntryPoint {

    private final JsonMapper json;

    public ApiAuthenticationEntryPoint(JsonMapper json) {
        this.json = json;
    }

    @Override
    public void commence(HttpServletRequest request, HttpServletResponse response, AuthenticationException e) throws IOException {
        if (e instanceof InvalidBearerTokenException) {
            write(request, response, ErrorCode.TOKEN_EXPIRED, "invalid or expired access token");
        } else {
            write(request, response, ErrorCode.UNAUTHORIZED, "authentication required");
        }
    }

    void write(HttpServletRequest request, HttpServletResponse response, ErrorCode code, String message) throws IOException {
        response.setStatus(code.status().value());
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        json.writeValue(response.getOutputStream(), ErrorResponse.of(code, message, RequestIdFilter.current(request)));
    }
}
