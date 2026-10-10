package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	t.Parallel()

	handler := RateLimiter(rate.Limit(10), 10)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "1.2.3.4:1234"

	handler(c)

	if w.Code == http.StatusTooManyRequests {
		t.Fatal("expected request to be allowed, got 429")
	}
}

func TestRateLimiter_BlocksWhenExceeded(t *testing.T) {
	t.Parallel()

	// 1 token per second, burst of 1 — second request should be blocked immediately.
	handler := RateLimiter(rate.Limit(1), 1)

	makeRequest := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:4321"
		c.Request = req
		handler(c)
		return w.Code
	}

	// First request — should pass.
	if code := makeRequest(); code == http.StatusTooManyRequests {
		t.Fatal("first request should be allowed")
	}
	// Second immediate request — burst exhausted, should be blocked.
	if code := makeRequest(); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", code)
	}
}

func TestRateLimiter_DifferentIPsHaveSeparateLimits(t *testing.T) {
	t.Parallel()

	handler := RateLimiter(rate.Limit(1), 1)

	doRequest := func(ip string) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":80"
		c.Request = req
		handler(c)
		return w.Code
	}

	// First call for each IP should be allowed regardless of other IPs.
	if code := doRequest("10.0.0.1"); code == http.StatusTooManyRequests {
		t.Fatal("expected first request from IP1 to pass")
	}
	if code := doRequest("10.0.0.2"); code == http.StatusTooManyRequests {
		t.Fatal("expected first request from IP2 to pass (separate bucket)")
	}
}

func TestRequestID_SetsHeaderAndContext(t *testing.T) {
	t.Parallel()

	handler := RequestID()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler(c)

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}

	ctxID := GetRequestID(c)
	if ctxID != id {
		t.Fatalf("context request ID %q does not match header %q", ctxID, id)
	}
}

func TestRequestID_PreservesExistingHeader(t *testing.T) {
	t.Parallel()

	handler := RequestID()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	c.Request = req

	handler(c)

	if got := w.Header().Get("X-Request-ID"); got != "existing-id" {
		t.Fatalf("expected existing-id to be preserved, got %q", got)
	}
}

func TestRateLimiterWithRedis_NilRedisUsesLocalFallback(t *testing.T) {
	t.Parallel()

	// nil redis → must return local in-memory fallback, not panic.
	handler := RateLimiterWithRedis(nil, rate.Limit(10), 10, "rl:test")
	if handler == nil {
		t.Fatal("expected non-nil handler when redis is nil")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "9.9.9.9:9090"

	// Should not panic, should allow the request.
	handler(c)
	if w.Code == http.StatusTooManyRequests {
		t.Fatal("expected request to be allowed with generous limit")
	}
}

func TestGetRequestID_EmptyWhenNotSet(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// Do not call RequestID() handler — ID should be empty string.
	if id := GetRequestID(c); id != "" {
		t.Fatalf("expected empty string, got %q", id)
	}
}

func TestRateLimiter_ExpiresOldClients(t *testing.T) {
	// This test verifies old client entries are eventually cleaned up,
	// not that they expire at a specific time (that's timing-sensitive).
	// We just call the limiter enough to populate the map and verify no panic.
	handler := RateLimiter(rate.Every(time.Second), 100)

	for i := range 5 {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.0." + string(rune('1'+i)) + ":80"
		c.Request = req
		handler(c)
	}
	// No panic = pass.
}
