package api

import (
	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
)

func currentUserID(c *gin.Context) (int64, bool) {
	payload, ok := c.Get(middleware.AuthorizationPayloadKey)
	if !ok {
		return 0, false
	}
	authPayload, ok := payload.(*token.Payload)
	if !ok {
		return 0, false
	}
	return authPayload.UserID, true
}

// requireUserID writes a 401 and returns false when the request is anonymous.
func requireUserID(c *gin.Context) (int64, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, apperr.Unauthorized("authentication required"))
		return 0, false
	}
	return userID, true
}

// optionalViewerID returns the caller's ID, or nil for anonymous requests.
func optionalViewerID(c *gin.Context) *int64 {
	if id, ok := currentUserID(c); ok {
		return &id
	}
	return nil
}
