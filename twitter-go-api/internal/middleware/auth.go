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
)

func AuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := resolveAccessToken(c)
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			return
		}

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
			return
		}

		c.Set(AuthorizationPayloadKey, payload)
		c.Next()
	}
}

// OptionalAuthMiddleware decodes access tokens when present, but never blocks the request.
func OptionalAuthMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := resolveAccessToken(c)
		if err != nil {
			c.Next()
			return
		}

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err == nil {
			c.Set(AuthorizationPayloadKey, payload)
		}

		c.Next()
	}
}

func resolveAccessToken(c *gin.Context) (string, error) {
	authorizationHeader := c.GetHeader(authorizationHeaderKey)
	if len(authorizationHeader) == 0 {
		return "", errors.New("authorization header is not provided")
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) >= 2 && strings.ToLower(fields[0]) == authorizationTypeBearer {
		return fields[1], nil
	}

	return "", errors.New("invalid authorization header format")
}
