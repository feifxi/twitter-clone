package com.fei.twitter.shared.error;

import com.fasterxml.jackson.annotation.JsonInclude;
import java.util.List;

/** The body of every error response: {@code {code, message, requestId?, details?}}. */
@JsonInclude(JsonInclude.Include.NON_EMPTY)
public record ErrorResponse(String code, String message, String requestId, List<FieldError> details) {

    public record FieldError(String field, String message) {}

    public static ErrorResponse of(ErrorCode code, String message, String requestId) {
        return new ErrorResponse(code.name(), message, requestId, null);
    }
}
