package com.fei.twitter.auth;

import com.fei.twitter.security.SessionCookies;
import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.ratelimit.RateLimit;
import com.fei.twitter.shared.web.SuccessResponse;
import com.fei.twitter.user.CurrentUserResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Auth")
@RestController
@RequestMapping("/api/v1/auth")
class AuthController {

    private final AuthService auth;
    private final SessionCookies cookies;

    AuthController(AuthService auth, SessionCookies cookies) {
        this.auth = auth;
        this.cookies = cookies;
    }

    record GoogleLoginRequest(@NotBlank(message = "this field is required") String idToken) {}

    @Operation(summary = "Exchange a Google ID token for a session (httpOnly cookies)")
    @RateLimit(RateLimit.Policy.AUTH)
    @PostMapping("/google")
    CurrentUserResponse loginGoogle(@Valid @RequestBody GoogleLoginRequest request, HttpServletResponse response) {
        var session = auth.loginWithGoogle(request.idToken());
        cookies.set(response, session.accessToken(), session.accessExpiresAt(), session.refreshToken(), session.refreshExpiresAt());
        return CurrentUserResponse.of(session.user());
    }

    @Operation(summary = "Rotate the session in the refresh_token cookie")
    @RateLimit(RateLimit.Policy.AUTH)
    @PostMapping("/refresh")
    CurrentUserResponse refresh(HttpServletRequest request, HttpServletResponse response) {
        String refreshToken = SessionCookies.read(request, SessionCookies.REFRESH);
        if (refreshToken == null) {
            throw ApiException.unauthorized("missing refresh token");
        }
        // On failure the cookies are left alone: when two tabs refresh at once, the loser's
        // 401 may arrive after the winner's response, and clearing would sign the user out.
        var session = auth.refresh(refreshToken);
        cookies.set(response, session.accessToken(), session.accessExpiresAt(), session.refreshToken(), session.refreshExpiresAt());
        return CurrentUserResponse.of(session.user());
    }

    @Operation(summary = "End this browser's session and clear its cookies")
    @RateLimit(RateLimit.Policy.AUTH)
    @PostMapping("/logout")
    SuccessResponse logout(HttpServletRequest request, HttpServletResponse response) {
        auth.logout(SessionCookies.read(request, SessionCookies.REFRESH));
        cookies.clear(response);
        return SuccessResponse.OK;
    }

    @Operation(summary = "The signed-in user's own profile")
    @GetMapping("/me")
    CurrentUserResponse me(Viewer viewer) {
        return CurrentUserResponse.of(auth.me(viewer.requireId()));
    }
}
