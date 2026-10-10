package com.fei.twitter.auth;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.Optional;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/** One row per signed-in device; tokens are stored as SHA-256 hashes. */
@Repository
class RefreshTokenRepository {

    record StoredToken(long userId, Instant expiresAt) {}

    private final JdbcClient jdbc;

    RefreshTokenRepository(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    void create(long userId, String tokenHash, Instant expiresAt) {
        jdbc.sql("INSERT INTO refresh_tokens (user_id, token_hash, expiry_date) VALUES (:user, :hash, :expires)")
                .param("user", userId)
                .param("hash", tokenHash)
                .param("expires", OffsetDateTime.ofInstant(expiresAt, ZoneOffset.UTC))
                .update();
    }

    /**
     * Deletes and returns the token in one statement, so a refresh token works at most
     * once even when two requests race with it.
     */
    Optional<StoredToken> consume(String tokenHash) {
        return jdbc.sql("DELETE FROM refresh_tokens WHERE token_hash = :hash RETURNING user_id, expiry_date")
                .param("hash", tokenHash)
                .query((rs, n) -> new StoredToken(rs.getLong("user_id"), rs.getObject("expiry_date", OffsetDateTime.class).toInstant()))
                .optional();
    }

    void delete(String tokenHash) {
        jdbc.sql("DELETE FROM refresh_tokens WHERE token_hash = :hash").param("hash", tokenHash).update();
    }

    void deleteExpired(long userId) {
        jdbc.sql("DELETE FROM refresh_tokens WHERE user_id = :user AND expiry_date < now()").param("user", userId).update();
    }
}
