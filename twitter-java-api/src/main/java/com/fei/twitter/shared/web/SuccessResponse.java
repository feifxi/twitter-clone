package com.fei.twitter.shared.web;

public record SuccessResponse(boolean success) {
    public static final SuccessResponse OK = new SuccessResponse(true);
}
