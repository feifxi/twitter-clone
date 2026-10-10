package com.fei.twitter.shared.web;

import com.fasterxml.jackson.annotation.JsonInclude;
import java.util.List;
import java.util.function.Function;

/** The envelope of every paginated list. {@code nextCursor} is opaque and omitted on the last page. */
public record PageResponse<T>(
        List<T> items, boolean hasNext, @JsonInclude(JsonInclude.Include.NON_NULL) String nextCursor) {

    public static <T, R> PageResponse<R> of(Page<T> page, Function<T, R> toResponse) {
        return new PageResponse<>(
                page.items().stream().map(toResponse).toList(),
                page.next() != null,
                page.next() != null ? page.next().encode() : null);
    }
}
