package com.fei.twitter.shared.error;

/**
 * An error whose message is safe to show to clients. Services throw these
 * (never HTTP status codes); {@link GlobalExceptionHandler} turns them into responses.
 */
public class ApiException extends RuntimeException {

    private final ErrorCode code;

    public ApiException(ErrorCode code, String message) {
        super(message);
        this.code = code;
    }

    public ErrorCode code() {
        return code;
    }

    public static ApiException badRequest(String message) {
        return new ApiException(ErrorCode.BAD_REQUEST, message);
    }

    public static ApiException unauthorized(String message) {
        return new ApiException(ErrorCode.UNAUTHORIZED, message);
    }

    public static ApiException forbidden(String message) {
        return new ApiException(ErrorCode.FORBIDDEN, message);
    }

    public static ApiException notFound(String what) {
        return new ApiException(ErrorCode.NOT_FOUND, what + " not found");
    }

    public static ApiException conflict(String message) {
        return new ApiException(ErrorCode.CONFLICT, message);
    }
}
