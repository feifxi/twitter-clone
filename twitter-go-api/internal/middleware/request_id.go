package middleware

import (
	"regexp"

	"github.com/chanombude/twitter-go-api/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// Incoming IDs end up in logs, so only short, plain tokens are accepted.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// RequestID ensures every request has a correlation ID: it is echoed in the
// response header and stored on the request context for logging.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}

		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return logger.RequestID(c.Request.Context())
}
