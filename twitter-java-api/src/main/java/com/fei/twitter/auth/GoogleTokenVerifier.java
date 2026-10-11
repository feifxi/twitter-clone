package com.fei.twitter.auth;

/** Verifies Google ID tokens; replaced with a fake in tests. */
public interface GoogleTokenVerifier {

    record GoogleIdentity(String email, boolean emailVerified, String name, String picture) {}

    /** @throws com.fei.twitter.shared.error.ApiException UNAUTHORIZED when the token isn't valid for this app */
    GoogleIdentity verify(String idToken);
}
