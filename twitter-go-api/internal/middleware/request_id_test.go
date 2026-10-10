package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddlewareGeneratesAndSetsHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/ping", func(c *gin.Context) {
		if got := GetRequestID(c); got == "" {
			t.Fatal("expected request id in context")
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestRequestIDMiddlewareRespectsIncomingHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get(RequestIDHeader); got != "req-123" {
		t.Fatalf("expected propagated request id req-123, got %q", got)
	}
}
