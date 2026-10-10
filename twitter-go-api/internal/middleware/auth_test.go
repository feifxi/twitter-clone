package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req
	return c
}

func TestResolveAccessTokenFromHeader(t *testing.T) {
	t.Parallel()

	c := newTestContext()
	c.Request.Header.Set("Authorization", "Bearer header-token")

	got, err := resolveAccessToken(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "header-token" {
		t.Fatalf("expected header token, got %q", got)
	}
}

func TestResolveAccessTokenInvalidHeader(t *testing.T) {
	t.Parallel()

	c := newTestContext()
	c.Request.Header.Set("Authorization", "Token invalid")

	if _, err := resolveAccessToken(c); err == nil {
		t.Fatal("expected invalid header error")
	}
}
