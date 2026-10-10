package com.fei.twitter.shared.error;

import org.springframework.http.HttpStatus;

/** The API's error codes, shared with twitter-go-api (see docs/api-contract.md). */
public enum ErrorCode {
    BAD_REQUEST(HttpStatus.BAD_REQUEST),
    VALIDATION_ERROR(HttpStatus.BAD_REQUEST),
    UNAUTHORIZED(HttpStatus.UNAUTHORIZED),
    TOKEN_EXPIRED(HttpStatus.UNAUTHORIZED),
    FORBIDDEN(HttpStatus.FORBIDDEN),
    NOT_FOUND(HttpStatus.NOT_FOUND),
    CONFLICT(HttpStatus.CONFLICT),
    TOO_MANY_REQUESTS(HttpStatus.TOO_MANY_REQUESTS),
    INTERNAL_ERROR(HttpStatus.INTERNAL_SERVER_ERROR);

    private final HttpStatus status;

    ErrorCode(HttpStatus status) {
        this.status = status;
    }

    public HttpStatus status() {
        return status;
    }
}
