package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/realtime"
	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	srv, err := New(Config{AllowedOrigins: []string{"http://localhost:3000"}}, Services{}, nil, realtime.NewHub(nil), fakePinger{}, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv.Handler()
}

// Registering every route catches conflicts (e.g. static /users/me next to /users/:id) at test time, not at boot.
func TestRoutesRegisterWithoutConflicts(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /healthz, got %d", rec.Code)
	}
}

func TestNewRejectsMalformedOrigins(t *testing.T) {
	if _, err := New(Config{AllowedOrigins: []string{"localhost:3000/app"}}, Services{}, nil, realtime.NewHub(nil), fakePinger{}, nil); err == nil {
		t.Fatal("expected an error for an origin that isn't scheme://host")
	}
}

func TestCrossOriginProtection(t *testing.T) {
	srv, err := New(Config{AllowedOrigins: []string{"http://localhost:3000"}}, Services{}, nil, realtime.NewHub(nil), fakePinger{}, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	// Only this middleware: CORS also rejects unknown origins, which would hide what this one does.
	r := gin.New()
	r.Any("/probe", srv.crossOriginProtection, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	cases := map[string]struct {
		method  string
		headers map[string]string
		refused bool
	}{
		"cross-site form post":       {http.MethodPost, map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}, true},
		"untrusted origin, no fetch": {http.MethodDelete, map[string]string{"Origin": "https://evil.example"}, true},
		"the web app":                {http.MethodPost, map[string]string{"Origin": "http://localhost:3000", "Sec-Fetch-Site": "same-site"}, false},
		"same origin (Swagger UI)":   {http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, false},
		"non-browser client":         {http.MethodPost, nil, false},
		"cross-site read":            {http.MethodGet, map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/probe", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if refused := rec.Code == http.StatusForbidden; refused != tc.refused {
				t.Fatalf("refused=%v, want %v (status %d, body %s)", refused, tc.refused, rec.Code, rec.Body.String())
			}
		})
	}
}
