package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/middleware"
)

func newSession() auth.Session {
	now := time.Now()
	return auth.Session{
		User:             domain.User{ID: 9, Username: "tester", Email: "tester@example.com"},
		AccessToken:      "a-new",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:     "r-new",
		RefreshExpiresAt: now.Add(30 * 24 * time.Hour),
	}
}

// responseCookies indexes the Set-Cookie headers of a response by name.
func responseCookies(header http.Header) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range (&http.Response{Header: header}).Cookies() {
		out[c.Name] = c
	}
	return out
}

func TestRefreshRotatesSessionCookies(t *testing.T) {
	t.Parallel()

	fake := &fakeAuth{refreshFn: func(_ context.Context, refreshToken string) (auth.Session, error) {
		if refreshToken != "r-old" {
			t.Fatalf("expected the cookie's refresh token, got %q", refreshToken)
		}
		return newSession(), nil
	}}

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/refresh", nil, "")
	ctx.Request.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: "r-old"})
	s := &Server{cfg: Config{Production: true, CookieDomain: "example.com"}, svc: Services{Auth: fake}}
	s.refreshToken(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	cookies := responseCookies(rec.Header())
	access, refresh := cookies[middleware.AccessTokenCookie], cookies[middleware.RefreshTokenCookie]
	if access == nil || refresh == nil {
		t.Fatalf("expected both session cookies, got %v", rec.Header().Values("Set-Cookie"))
	}
	if access.Value != "a-new" || refresh.Value != "r-new" {
		t.Fatalf("unexpected cookie values: %q %q", access.Value, refresh.Value)
	}
	for _, c := range []*http.Cookie{access, refresh} {
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Domain != "example.com" || c.Path != "/" {
			t.Fatalf("cookie %s has unsafe attributes: %+v", c.Name, c)
		}
	}
	if access.MaxAge <= 0 || access.MaxAge > int((15*time.Minute).Seconds()) {
		t.Fatalf("access cookie must live as long as the access token, got MaxAge=%d", access.MaxAge)
	}
	if refresh.MaxAge <= access.MaxAge {
		t.Fatalf("refresh cookie must outlive the access cookie, got %d <= %d", refresh.MaxAge, access.MaxAge)
	}

	// Tokens never appear in the body, where page scripts could read them.
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["username"] != "tester" || body["accessToken"] != nil || body["refreshToken"] != nil {
		t.Fatalf("expected the current user without tokens, got %v", body)
	}
}

func TestRefreshFailureLeavesCookiesAlone(t *testing.T) {
	t.Parallel()

	fake := &fakeAuth{refreshFn: func(context.Context, string) (auth.Session, error) {
		return auth.Session{}, apperr.Unauthorized("refresh token not found or revoked")
	}}

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/refresh", nil, "")
	ctx.Request.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: "r-used"})
	(&Server{svc: Services{Auth: fake}}).refreshToken(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if got := rec.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("a failed refresh must not touch cookies (a concurrent refresh may have renewed them), got %v", got)
	}
}

func TestLogoutEndsTheCookieSessionAndClearsCookies(t *testing.T) {
	t.Parallel()

	var got string
	fake := &fakeAuth{logoutFn: func(_ context.Context, refreshToken string) error {
		got = refreshToken
		return nil
	}}

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/logout", nil, "")
	ctx.Request.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: "r-1"})
	(&Server{svc: Services{Auth: fake}}).logout(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got != "r-1" {
		t.Fatalf("expected the cookie's refresh token to be revoked, got %q", got)
	}
	cookies := responseCookies(rec.Header())
	for _, name := range []string{middleware.AccessTokenCookie, middleware.RefreshTokenCookie} {
		if c := cookies[name]; c == nil || c.MaxAge >= 0 || c.Value != "" {
			t.Fatalf("expected %s to be deleted, got %+v", name, c)
		}
	}
}
