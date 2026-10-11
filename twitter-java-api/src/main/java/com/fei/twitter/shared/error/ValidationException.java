package com.fei.twitter.shared.error;

/** A single invalid input field, reported as VALIDATION_ERROR with details. */
public class ValidationException extends ApiException {

    private final String field;

    public ValidationException(String field, String message) {
        super(ErrorCode.VALIDATION_ERROR, message);
        this.field = field;
    }

    public String field() {
        return field;
    }
}
