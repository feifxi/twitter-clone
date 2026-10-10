package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"

	// Session cookies set by the auth handlers. Browsers use these; other
	// clients (curl, tests) can send the access token as a Bearer header instead.
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"

	// CodeTokenExpired tells clients the credentials they sent are stale and a
	// POST /auth/refresh should fix it. Plain UNAUTHORIZED means none were sent.
	CodeTokenExpired = "TOKEN_EXPIRED"
)

// AuthMiddleware requires a valid access token.
func AuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authenticate(c, tokenMaker) {
			if !c.IsAborted() { // no credentials at all; stale ones were already answered
				abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			}
			return
		}
		c.Next()
	}
}

// OptionalAuthMiddleware serves anonymous requests, and identifies the user when
// they're signed in. A signed-in user with an expired session gets 401 rather
// than silently anonymous data (e.g. tweets shown as not liked), so the client
// knows to refresh.
func OptionalAuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticate(c, tokenMaker)
		c.Next()
	}
}

// authenticate stores the token payload on the context and reports whether the
// request is authenticated. When the request carries credentials that are no
// longer valid, it aborts with TOKEN_EXPIRED: an expired or invalid access token,
// or only a refresh cookie (the browser drops the access cookie when it expires).
func authenticate(c *gin.Context, tokenMaker token.Maker) bool {
	accessToken, err := resolveAccessToken(c)
	if err != nil {
		if refresh, _ := c.Cookie(RefreshTokenCookie); refresh != "" {
			abortWithError(c, http.StatusUnauthorized, CodeTokenExpired, "access token expired")
		}
		return false
	}

	payload, err := tokenMaker.VerifyToken(accessToken)
	if err != nil {
		abortWithError(c, http.StatusUnauthorized, CodeTokenExpired, "invalid or expired access token")
		return false
	}

	c.Set(AuthorizationPayloadKey, payload)
	return true
}

// resolveAccessToken prefers an explicit Authorization header over the session cookie.
func resolveAccessToken(c *gin.Context) (string, error) {
	authorizationHeader := c.GetHeader(authorizationHeaderKey)
	if len(authorizationHeader) == 0 {
		if cookie, err := c.Cookie(AccessTokenCookie); err == nil && cookie != "" {
			return cookie, nil
		}
		return "", errors.New("no access token")
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) >= 2 && strings.ToLower(fields[0]) == authorizationTypeBearer {
		return fields[1], nil
	}

	return "", errors.New("invalid authorization header format")
}
