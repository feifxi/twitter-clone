package com.fei.twitter.auth;

import com.fei.twitter.config.AppProperties;
import com.fei.twitter.security.TokenService;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.user.User;
import com.fei.twitter.user.UserRepository;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.time.Clock;
import java.time.Instant;
import java.util.HexFormat;
import java.util.Locale;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/** Google sign-in and session management (short-lived access JWT + rotating refresh token). */
@Service
public class AuthService {

    private static final int MAX_DISPLAY_NAME = 50;
    private static final int USERNAME_BASE_LENGTH = 11; // + "_" + 8 random characters = 20, the column limit
    private static final String ALPHABET = "abcdefghijklmnopqrstuvwxyz234567";
    private static final SecureRandom RANDOM = new SecureRandom();

    private final GoogleTokenVerifier google;
    private final UserRepository users;
    private final RefreshTokenRepository refreshTokens;
    private final TokenService tokens;
    private final AppProperties props;
    private final Clock clock;

    AuthService(GoogleTokenVerifier google, UserRepository users, RefreshTokenRepository refreshTokens,
                TokenService tokens, AppProperties props, Clock clock) {
        this.google = google;
        this.users = users;
        this.refreshTokens = refreshTokens;
        this.tokens = tokens;
        this.props = props;
        this.clock = clock;
    }

    /** A signed-in device. */
    public record Session(User user, String accessToken, Instant accessExpiresAt, String refreshToken, Instant refreshExpiresAt) {}

    @Transactional
    public Session loginWithGoogle(String idToken) {
        var identity = google.verify(idToken);
        if (identity.email() == null || identity.email().isBlank()) {
            throw ApiException.unauthorized("invalid google token");
        }
        // Accounts are keyed by email, so an unverified email would let someone claim another person's account.
        if (!identity.emailVerified()) {
            throw ApiException.unauthorized("google email is not verified");
        }

        User user = users.findByEmail(identity.email()).orElseGet(() -> users.save(new User(
                generatedUsername(identity.email()),
                identity.email(),
                truncate(identity.name() == null ? "" : identity.name().strip(), MAX_DISPLAY_NAME),
                identity.picture(),
                "GOOGLE")));
        return issueSession(user);
    }

    /** Rotates the session: the presented refresh token is consumed and a new pair issued. */
    @Transactional
    public Session refresh(String refreshToken) {
        var stored = refreshTokens.consume(hash(refreshToken))
                .orElseThrow(() -> ApiException.unauthorized("refresh token not found or revoked"));
        if (clock.instant().isAfter(stored.expiresAt())) {
            throw ApiException.unauthorized("refresh token expired");
        }
        User user = users.findById(stored.userId()).orElseThrow(() -> ApiException.notFound("user"));
        return issueSession(user);
    }

    /** Ends this device's session; other devices stay signed in. */
    @Transactional
    public void logout(String refreshToken) {
        if (refreshToken != null && !refreshToken.isEmpty()) {
            refreshTokens.delete(hash(refreshToken));
        }
    }

    @Transactional(readOnly = true)
    public User me(long userId) {
        return users.findById(userId).orElseThrow(() -> ApiException.notFound("user"));
    }

    private Session issueSession(User user) {
        var access = tokens.issue(user.getId());
        String refreshToken = randomToken();
        Instant refreshExpiresAt = clock.instant().plus(props.refreshTokenTtl());
        refreshTokens.create(user.getId(), hash(refreshToken), refreshExpiresAt);
        refreshTokens.deleteExpired(user.getId());
        return new Session(user, access.value(), access.expiresAt(), refreshToken, refreshExpiresAt);
    }

    /** A placeholder handle from the email; users pick their own during onboarding. */
    static String generatedUsername(String email) {
        String local = email.strip().toLowerCase(Locale.ROOT).split("@", 2)[0].replaceAll("[^a-z0-9_]", "");
        String base = local.isEmpty() ? "user" : truncate(local, USERNAME_BASE_LENGTH);
        return base + "_" + randomString(8);
    }

    private static String randomToken() {
        return randomString(26);
    }

    private static String randomString(int length) {
        StringBuilder sb = new StringBuilder(length);
        for (int i = 0; i < length; i++) {
            sb.append(ALPHABET.charAt(RANDOM.nextInt(ALPHABET.length())));
        }
        return sb.toString();
    }

    static String hash(String token) {
        try {
            byte[] digest = MessageDigest.getInstance("SHA-256").digest(token.getBytes(StandardCharsets.UTF_8));
            return HexFormat.of().formatHex(digest);
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException(e);
        }
    }

    private static String truncate(String s, int maxCodePoints) {
        return s.codePointCount(0, s.length()) <= maxCodePoints ? s : s.substring(0, s.offsetByCodePoints(0, maxCodePoints));
    }
}
