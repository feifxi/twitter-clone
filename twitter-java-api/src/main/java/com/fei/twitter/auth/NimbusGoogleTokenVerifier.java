package com.fei.twitter.auth;

import com.fei.twitter.config.AppProperties;
import com.fei.twitter.shared.error.ApiException;
import java.util.List;
import java.util.Set;
import org.springframework.security.oauth2.core.DelegatingOAuth2TokenValidator;
import org.springframework.security.oauth2.core.OAuth2TokenValidator;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtClaimValidator;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtException;
import org.springframework.security.oauth2.jwt.JwtValidators;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;
import org.springframework.stereotype.Component;

/** Checks the signature against Google's published keys, the issuer, the audience (our client ID) and expiry. */
@Component
class NimbusGoogleTokenVerifier implements GoogleTokenVerifier {

    private static final String GOOGLE_KEYS = "https://www.googleapis.com/oauth2/v3/certs";
    private static final Set<String> GOOGLE_ISSUERS = Set.of("accounts.google.com", "https://accounts.google.com");

    private final JwtDecoder decoder;

    NimbusGoogleTokenVerifier(AppProperties props) {
        NimbusJwtDecoder decoder = NimbusJwtDecoder.withJwkSetUri(GOOGLE_KEYS).build();
        OAuth2TokenValidator<Jwt> issuer = new JwtClaimValidator<String>("iss", GOOGLE_ISSUERS::contains);
        OAuth2TokenValidator<Jwt> audience =
                new JwtClaimValidator<List<String>>("aud", aud -> aud != null && aud.contains(props.googleClientId()));
        decoder.setJwtValidator(new DelegatingOAuth2TokenValidator<>(JwtValidators.createDefault(), issuer, audience));
        this.decoder = decoder;
    }

    @Override
    public GoogleIdentity verify(String idToken) {
        try {
            Jwt jwt = decoder.decode(idToken);
            return new GoogleIdentity(
                    jwt.getClaimAsString("email"),
                    Boolean.TRUE.equals(jwt.getClaimAsBoolean("email_verified")),
                    jwt.getClaimAsString("name"),
                    jwt.getClaimAsString("picture"));
        } catch (JwtException e) {
            throw ApiException.unauthorized("invalid google token");
        }
    }
}
