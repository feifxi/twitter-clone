package com.fei.twitter.shared.web;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;
import java.util.Base64;
import tools.jackson.databind.DeserializationFeature;
import tools.jackson.databind.json.JsonMapper;

/**
 * An opaque keyset cursor: the sort key of the last item a client has seen. Encoded
 * like the Go API's (base64url JSON with short keys); clients must treat it as opaque.
 * Unused fields are left at their defaults and omitted.
 */
@JsonInclude(JsonInclude.Include.NON_DEFAULT)
public record Cursor(
        @JsonProperty("t") Instant time,
        @JsonProperty("i") long id,
        @JsonProperty("k") int kind,
        @JsonProperty("s") double score,
        @JsonProperty("a") Instant anchor,
        @JsonProperty("o") int offset) {

    // Omitted fields decode to their defaults (Jackson 3 rejects missing primitives by default).
    private static final JsonMapper JSON = JsonMapper.builder()
            .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
            .build();

    public static Cursor ofTime(Instant time, long id) {
        return new Cursor(time, id, 0, 0, null, 0);
    }

    public static Cursor ofTimeline(Instant time, int kind, long id) {
        return new Cursor(time, id, kind, 0, null, 0);
    }

    public static Cursor ofScore(Instant anchor, double score, long id) {
        return new Cursor(null, id, 0, score, anchor, 0);
    }

    public static Cursor ofOffset(int offset) {
        return new Cursor(null, 0, 0, 0, null, offset);
    }

    public String encode() {
        return Base64.getUrlEncoder().withoutPadding().encodeToString(JSON.writeValueAsBytes(this));
    }

    /** @throws IllegalArgumentException when the cursor is malformed */
    public static Cursor decode(String encoded) {
        try {
            Cursor cursor = JSON.readValue(Base64.getUrlDecoder().decode(encoded), Cursor.class);
            if (cursor.offset < 0) {
                throw new IllegalArgumentException("negative offset");
            }
            return cursor;
        } catch (RuntimeException e) {
            throw new IllegalArgumentException("invalid cursor", e);
        }
    }
}
