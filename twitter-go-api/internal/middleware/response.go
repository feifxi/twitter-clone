package middleware

import (
	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func abortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, apperr.ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: GetRequestID(c),
	})
}
