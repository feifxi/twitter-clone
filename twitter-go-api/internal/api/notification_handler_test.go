package api

import (
	"net/http"
	"testing"
)

func TestGetUnreadNotificationCount_MissingAuth(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/notifications/unread-count", nil, "")

	s := &Server{}
	s.getUnreadNotificationCount(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMarkNotificationRead_MissingAuth(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/notifications/mark-read", nil, "")

	s := &Server{}
	s.markNotificationRead(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
