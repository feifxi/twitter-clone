package com.fei.twitter.shared.web;

import java.util.List;
import java.util.function.Function;

/** The envelope of every paginated list. {@code nextCursor} is opaque, and null on the last page. */
public record PageResponse<T>(List<T> items, boolean hasNext, String nextCursor) {

    public static <T, R> PageResponse<R> of(Page<T> page, Function<T, R> toResponse) {
        return new PageResponse<>(
                page.items().stream().map(toResponse).toList(),
                page.next() != null,
                page.next() != null ? page.next().encode() : null);
    }
}
