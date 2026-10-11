package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// SkipLogKey, set on the gin context, suppresses the access log line (used for bot noise on unknown routes).
const SkipLogKey = "skip_log"

// RequestLogger writes one access log line per request; probes are excluded.
// The query string is never logged: it can carry credentials.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		if skip, _ := c.Get(SkipLogKey); skip == true {
			return
		}

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		route := c.FullPath()
		if route == "" {
			route = path
		}

		slog.Log(c.Request.Context(), level, "http request",
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.String("ip", c.ClientIP()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("bytes", c.Writer.Size()),
		)
	}
}
