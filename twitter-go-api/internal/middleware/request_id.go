package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader     = "X-Request-ID"
	RequestIDContextKey = "request_id"
)

// RequestID ensures every request has a correlation id and returns it to the client.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDContextKey, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if raw, ok := c.Get(RequestIDContextKey); ok {
		if v, ok := raw.(string); ok {
			return v
		}
	}
	return ""
}
