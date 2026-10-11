package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewares(t *testing.T) {
	maker, err := token.NewJWTMaker("01234567890123456789012345678901")
	if err != nil {
		t.Fatal(err)
	}
	valid, err := maker.CreateToken(7, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := maker.CreateToken(7, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	ok := func(c *gin.Context) {
		_, signedIn := c.Get(AuthorizationPayloadKey)
		c.JSON(http.StatusOK, gin.H{"signedIn": signedIn})
	}
	r.GET("/required", AuthMiddleware(maker), ok)
	r.GET("/optional", OptionalAuthMiddleware(maker), ok)

	cases := map[string]struct {
		cookies  map[string]string
		path     string
		status   int
		code     string
		signedIn bool
	}{
		"anonymous, required":           {nil, "/required", 401, "UNAUTHORIZED", false},
		"anonymous, optional":           {nil, "/optional", 200, "", false},
		"valid cookie, required":        {map[string]string{AccessTokenCookie: valid}, "/required", 200, "", true},
		"valid cookie, optional":        {map[string]string{AccessTokenCookie: valid}, "/optional", 200, "", true},
		"expired cookie, required":      {map[string]string{AccessTokenCookie: expired}, "/required", 401, CodeTokenExpired, false},
		"expired cookie, optional":      {map[string]string{AccessTokenCookie: expired}, "/optional", 401, CodeTokenExpired, false},
		"only refresh cookie, required": {map[string]string{RefreshTokenCookie: "r"}, "/required", 401, CodeTokenExpired, false},
		"only refresh cookie, optional": {map[string]string{RefreshTokenCookie: "r"}, "/optional", 401, CodeTokenExpired, false},
		"garbage token, optional":       {map[string]string{AccessTokenCookie: "not-a-jwt"}, "/optional", 401, CodeTokenExpired, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			for k, v := range tc.cookies {
				req.AddCookie(&http.Cookie{Name: k, Value: v})
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.status {
				t.Fatalf("status %d, want %d (body %s)", rec.Code, tc.status, rec.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("expected one JSON body, got %q: %v", rec.Body.String(), err)
			}
			if tc.status != http.StatusOK && body["code"] != tc.code {
				t.Fatalf("code %v, want %s", body["code"], tc.code)
			}
			if tc.status == http.StatusOK && body["signedIn"] != tc.signedIn {
				t.Fatalf("signedIn %v, want %v", body["signedIn"], tc.signedIn)
			}
		})
	}
}
