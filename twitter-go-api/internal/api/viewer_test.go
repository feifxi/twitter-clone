package api

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
