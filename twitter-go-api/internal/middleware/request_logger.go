package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestLogger logs one line per request (health and metrics probes excluded).
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The query string is never logged: it can carry credentials (e.g. a WebSocket access_token).
		start := time.Now()
		path := c.Request.URL.Path

		// Skip internal health and metrics checks to reduce log noise
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		route := c.FullPath()
		if route == "" {
			route = path
		}
		requestID := GetRequestID(c)

		c.Next()

		// Skip logging if a middleware (like NoRoute) has flagged it
		if skip, _ := c.Get("skip_log"); skip == true {
			return
		}

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
		bodySize := c.Writer.Size()

		event := log.Info()
		if statusCode >= 400 {
			event = log.Error()
		}

		event.
			Int("status", statusCode).
			Str("method", method).
			Str("path", path).
			Str("route", route).
			Str("request_id", requestID).
			Str("ip", clientIP).
			Str("user_agent", userAgent).
			Dur("duration", duration).
			Int("body_size", bodySize).
			Str("error", errorMessage).
			Msg("HTTP request")
	}
}
