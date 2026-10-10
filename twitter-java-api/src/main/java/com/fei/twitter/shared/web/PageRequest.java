package com.fei.twitter.shared.web;

/**
 * A page of a keyset-paginated list: the items after {@code after} (null = first page).
 * Resolved from {@code ?cursor=&size=} by {@link PageRequestArgumentResolver}.
 */
public record PageRequest(Cursor after, int limit) {

    public static final int DEFAULT_LIMIT = 20;
    public static final int MAX_LIMIT = 50;

    public static PageRequest first(int limit) {
        return new PageRequest(null, limit);
    }

    /** Rows to query: one extra reveals whether another page exists. */
    public int fetchLimit() {
        return limit + 1;
    }

    public int offset() {
        return after == null ? 0 : after.offset();
    }
}
