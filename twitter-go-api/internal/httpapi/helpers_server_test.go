package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOptionalViewerID_AuthenticatedReturnsPointer(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	setAuthorizedUser(ctx, 42)

	id := optionalViewerID(ctx)
	if id == nil {
		t.Fatal("expected non-nil viewer ID for authenticated user")
	}
	if *id != 42 {
		t.Fatalf("expected viewer ID 42, got %d", *id)
	}
}

func TestOptionalViewerID_UnauthenticatedReturnsNil(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if id := optionalViewerID(ctx); id != nil {
		t.Fatalf("expected nil for unauthenticated user, got %d", *id)
	}
}

func TestParseLimit_DefaultWhenAbsent(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10, got %d", got)
	}
}

func TestParseLimit_ParsesValidValue(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=25", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
}

func TestParseLimit_CapsAtMax(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=999", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 50 {
		t.Fatalf("expected cap at 50, got %d", got)
	}
}

func TestParseLimit_InvalidFallsBackToDefault(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10 on invalid input, got %d", got)
	}
}

func TestParseLimit_ZeroFallsBackToDefault(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=0", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10 for limit=0, got %d", got)
	}
}
