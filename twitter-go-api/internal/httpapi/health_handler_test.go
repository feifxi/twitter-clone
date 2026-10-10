package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestReadyz(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		db         fakePinger
		wantStatus int
		wantCheck  string
	}{
		"database up":   {fakePinger{}, http.StatusOK, "ok"},
		"database down": {fakePinger{errors.New("connection refused")}, http.StatusServiceUnavailable, "error"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c, rec := newHandlerTestContext(http.MethodGet, "/readyz", nil, "")

			(&Server{db: tc.db}).readyz(c)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			var body HealthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Checks["database"] != tc.wantCheck {
				t.Fatalf("expected database=%s, got %+v", tc.wantCheck, body.Checks)
			}
			if strings.Contains(rec.Body.String(), "connection refused") {
				t.Fatalf("readiness response must not leak error details: %s", rec.Body.String())
			}
		})
	}
}
