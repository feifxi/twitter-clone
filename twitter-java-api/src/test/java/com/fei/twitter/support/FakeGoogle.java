package com.fei.twitter.support;

import com.fei.twitter.auth.GoogleTokenVerifier;
import com.fei.twitter.shared.error.ApiException;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Primary;

/** Accepts ID tokens of the form {@code google:<email>} (and {@code unverified:<email>}). */
@TestConfiguration(proxyBeanMethods = false)
public class FakeGoogle {

    @Bean
    @Primary
    GoogleTokenVerifier fakeGoogleTokenVerifier() {
        return idToken -> {
            if (idToken.startsWith("google:")) {
                String email = idToken.substring("google:".length());
                return new GoogleTokenVerifier.GoogleIdentity(email, true, "Test " + email.split("@")[0], null);
            }
            if (idToken.startsWith("unverified:")) {
                return new GoogleTokenVerifier.GoogleIdentity(idToken.substring("unverified:".length()), false, "X", null);
            }
            throw ApiException.unauthorized("invalid google token");
        };
    }
}
