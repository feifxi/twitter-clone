package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/realtime"
)

// Registering every route catches conflicts (e.g. static /users/me next to /users/:id) at test time, not at boot.
func TestRoutesRegisterWithoutConflicts(t *testing.T) {
	srv := New(Config{AllowedOrigins: []string{"http://localhost:3000"}}, Services{}, nil, realtime.NewHub(nil), fakePinger{}, nil)

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /healthz, got %d", rec.Code)
	}
}
