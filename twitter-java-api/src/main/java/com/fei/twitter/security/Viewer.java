package com.fei.twitter.security;

import com.fei.twitter.shared.error.ApiException;

/**
 * The caller of a request: a signed-in user, or anonymous on public endpoints.
 * Declare it as a controller parameter; {@link ViewerArgumentResolver} fills it in.
 */
public record Viewer(Long id) {

    public static final Viewer ANONYMOUS = new Viewer(null);

    public boolean signedIn() {
        return id != null;
    }

    /** The user ID; protected routes are already authenticated by SecurityConfig. */
    public long requireId() {
        if (id == null) {
            throw ApiException.unauthorized("authentication required");
        }
        return id;
    }
}
