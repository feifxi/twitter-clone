package com.fei.twitter.security;

import com.fei.twitter.shared.error.ErrorCode;
import com.fei.twitter.shared.error.ErrorResponse;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.web.cors.DefaultCorsProcessor;
import tools.jackson.databind.json.JsonMapper;

/** Rejects disallowed origins with the API's JSON error body instead of plain text. */
class JsonCorsProcessor extends DefaultCorsProcessor {

    private final JsonMapper json;

    JsonCorsProcessor(JsonMapper json) {
        this.json = json;
    }

    @Override
    protected void rejectRequest(ServerHttpResponse response) throws IOException {
        response.setStatusCode(HttpStatus.FORBIDDEN);
        response.getHeaders().setContentType(MediaType.APPLICATION_JSON);
        byte[] body = json.writeValueAsString(ErrorResponse.of(ErrorCode.FORBIDDEN, "origin not allowed", null))
                .getBytes(StandardCharsets.UTF_8);
        response.getBody().write(body);
        response.flush();
    }
}
