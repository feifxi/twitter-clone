package com.fei.twitter.search;

import com.fei.twitter.shared.web.PageRequest;

/** {@code ?limit=} for small unpaginated lists: invalid values fall back to the default; capped at 50. */
public final class Limits {

    private Limits() {}

    public static int parse(String raw, int defaultValue) {
        try {
            int value = Integer.parseInt(raw.trim());
            return value < 1 ? defaultValue : Math.min(value, PageRequest.MAX_LIMIT);
        } catch (NumberFormatException e) {
            return defaultValue;
        }
    }
}
