package api

import (
	"net/http"
	"time"

	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Sessions live in two httpOnly cookies, so page scripts (and any XSS) can't
// read the tokens. The web app and the API share one origin, so the browser
// sends them on every API request and the Next.js server forwards them when it
// renders pages.
//
// SameSite=Lax keeps the cookies off cross-site subrequests; cross-origin
// writes are refused separately (see crossOriginProtection).

func (s *Server) setSessionCookies(c *gin.Context, session auth.Session) {
	s.setCookie(c, middleware.AccessTokenCookie, session.AccessToken, time.Until(session.AccessExpiresAt))
	s.setCookie(c, middleware.RefreshTokenCookie, session.RefreshToken, time.Until(session.RefreshExpiresAt))
}

func (s *Server) clearSessionCookies(c *gin.Context) {
	s.setCookie(c, middleware.AccessTokenCookie, "", -1)
	s.setCookie(c, middleware.RefreshTokenCookie, "", -1)
}

// setCookie writes a cookie that expires after maxAge; a negative maxAge deletes it.
func (s *Server) setCookie(c *gin.Context, name, value string, maxAge time.Duration) {
	seconds := int(maxAge.Seconds())
	if maxAge < 0 {
		seconds = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		MaxAge:   seconds,
		HttpOnly: true,
		Secure:   s.cfg.Production, // local dev runs on plain http://localhost
		SameSite: http.SameSiteLaxMode,
	})
}
