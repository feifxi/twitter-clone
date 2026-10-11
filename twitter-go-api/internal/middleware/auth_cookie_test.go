package middleware

import (
	"net/http"
	"testing"
)

func TestResolveAccessTokenFromCookie(t *testing.T) {
	t.Parallel()

	c := newTestContext()
	c.Request.AddCookie(&http.Cookie{Name: AccessTokenCookie, Value: "cookie-token"})

	got, err := resolveAccessToken(c)
	if err != nil || got != "cookie-token" {
		t.Fatalf("expected cookie token, got %q, %v", got, err)
	}
}

func TestResolveAccessTokenPrefersHeaderOverCookie(t *testing.T) {
	t.Parallel()

	c := newTestContext()
	c.Request.AddCookie(&http.Cookie{Name: AccessTokenCookie, Value: "cookie-token"})
	c.Request.Header.Set("Authorization", "Bearer header-token")

	if got, _ := resolveAccessToken(c); got != "header-token" {
		t.Fatalf("expected the explicit header to win, got %q", got)
	}
}

func TestResolveAccessTokenMissing(t *testing.T) {
	t.Parallel()

	if _, err := resolveAccessToken(newTestContext()); err == nil {
		t.Fatal("expected an error without header or cookie")
	}
}
