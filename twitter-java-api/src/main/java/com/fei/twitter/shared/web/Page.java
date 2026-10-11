package com.fei.twitter.shared.web;

import java.util.List;
import java.util.function.Function;

/** A page of results and the cursor for the next one (null on the last page). */
public record Page<T>(List<T> items, Cursor next) {

    /**
     * Builds a page from rows fetched with {@link PageRequest#fetchLimit()}: drops the probe
     * row and takes the next cursor from the last kept row.
     */
    public static <T> Page<T> fromRows(List<T> rows, PageRequest request, Function<T, Cursor> cursorOf) {
        if (rows.size() <= request.limit()) {
            return new Page<>(rows, null);
        }
        List<T> kept = rows.subList(0, request.limit());
        return new Page<>(List.copyOf(kept), cursorOf.apply(kept.getLast()));
    }

    public <R> Page<R> map(Function<T, R> mapper) {
        return new Page<>(items.stream().map(mapper).toList(), next);
    }
}
