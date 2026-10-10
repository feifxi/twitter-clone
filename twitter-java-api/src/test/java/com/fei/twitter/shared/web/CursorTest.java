package com.fei.twitter.shared.web;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.Base64;
import org.junit.jupiter.api.Test;

class CursorTest {

    @Test
    void roundTripsEveryKind() {
        Instant t = Instant.parse("2026-06-15T12:00:00.123456Z");
        for (Cursor c : new Cursor[] {Cursor.ofTime(t, 42), Cursor.ofTimeline(t, 1, 7), Cursor.ofScore(t, 3.25, 9), Cursor.ofOffset(40)}) {
            assertThat(Cursor.decode(c.encode())).isEqualTo(c);
        }
    }

    @Test
    void encodesLikeTheGoApiShortKeysOmittingDefaults() {
        String json = new String(Base64.getUrlDecoder().decode(Cursor.ofTime(Instant.parse("2026-06-15T12:00:00Z"), 42).encode()), StandardCharsets.UTF_8);
        assertThat(json).isEqualTo("{\"t\":\"2026-06-15T12:00:00Z\",\"i\":42}");
    }

    @Test
    void rejectsGarbageAndNegativeOffsets() {
        assertThatThrownBy(() -> Cursor.decode("not base64!")).isInstanceOf(IllegalArgumentException.class);
        String negative = Base64.getUrlEncoder().withoutPadding().encodeToString("{\"o\":-1}".getBytes(StandardCharsets.UTF_8));
        assertThatThrownBy(() -> Cursor.decode(negative)).isInstanceOf(IllegalArgumentException.class);
    }
}
