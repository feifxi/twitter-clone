package com.fei.twitter.config;

import jakarta.validation.constraints.AssertTrue;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.Size;
import java.net.URI;
import java.time.Duration;
import java.util.List;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.validation.annotation.Validated;

/** Application settings (bound from the Go-compatible environment variables in application.yml). */
@Validated
@ConfigurationProperties("app")
public record AppProperties(
        String environment,
        @NotEmpty List<String> allowedOrigins,
        String cookieDomain,
        @Size(min = 32, message = "TOKEN_SYMMETRIC_KEY must be at least 32 characters") String tokenSymmetricKey,
        Duration accessTokenTtl,
        Duration refreshTokenTtl,
        String googleClientId,
        String s3Bucket,
        String s3Region,
        String cloudfrontDomain,
        long maxMediaBytes,
        long maxAvatarBytes,
        long maxBannerBytes,
        String geminiApiKey,
        String geminiChatModel,
        String geminiEmbeddingModel,
        boolean enableRag,
        boolean rateLimitsEnabled) {

    public boolean production() {
        return "production".equalsIgnoreCase(environment);
    }

    /** Session cookies are Secure everywhere except local plain-HTTP development. */
    public boolean secureCookies() {
        return production();
    }

    public boolean geminiConfigured() {
        return geminiApiKey != null && !geminiApiKey.isBlank();
    }

    @AssertTrue(message = "ALLOWED_ORIGINS entries must look like https://host[:port]")
    boolean isAllowedOriginsWellFormed() {
        return allowedOrigins == null || allowedOrigins.stream().allMatch(AppProperties::isOrigin);
    }

    @AssertTrue(message = "production requires GOOGLE_CLIENT_ID, S3_BUCKET_NAME and CLOUDFRONT_DOMAIN, and no localhost origins")
    boolean isProductionReady() {
        if (!production()) {
            return true;
        }
        return notBlank(googleClientId) && notBlank(s3Bucket) && notBlank(cloudfrontDomain)
                && allowedOrigins.stream().noneMatch(o -> o.contains("localhost"));
    }

    private static boolean isOrigin(String origin) {
        try {
            URI uri = URI.create(origin.trim());
            String path = uri.getPath();
            return ("http".equals(uri.getScheme()) || "https".equals(uri.getScheme()))
                    && uri.getHost() != null
                    && (path == null || path.isEmpty() || path.equals("/"));
        } catch (IllegalArgumentException e) {
            return false;
        }
    }

    private static boolean notBlank(String s) {
        return s != null && !s.isBlank();
    }
}
