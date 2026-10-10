package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/realtime"
	"github.com/gin-gonic/gin"
)

const sseHeartbeatInterval = 15 * time.Second

// streamEvents godoc
// @Summary		Real-time Events (SSE)
// @Description	One Server-Sent Events stream per signed-in tab. Event names: `connected`, `notification` (NotificationResponse), `dm` (MessageResponse). Comment lines (`: ping`) are heartbeats. Reconnect on close; the server suggests a 3s retry.
// @Tags			Realtime
// @Produce			text/event-stream
// @Success		200	{string}	string	"text/event-stream"
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/events [get]
func (s *Server) streamEvents(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if _, ok := c.Writer.(http.Flusher); !ok {
		writeError(c, apperr.Internal("streaming unsupported", nil))
		return
	}

	events, unsubscribe := s.hub.Subscribe(userID)
	defer unsubscribe()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.WriteString("retry: 3000\n\n")
	writeSSE(c, "connected", gin.H{"status": "ok"})

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			// An SSE comment keeps proxies from closing an idle connection.
			_, _ = c.Writer.WriteString(": ping\n\n")
			c.Writer.Flush()
		case ev := <-events:
			writeRealtimeEvent(c, ev)
		}
	}
}

// writeRealtimeEvent maps hub payloads (domain types) to the public DTOs before writing.
func writeRealtimeEvent(c *gin.Context, ev realtime.Event) {
	ctx := c.Request.Context()
	switch ev.Type {
	case notification.EventType:
		var n domain.Notification
		if err := json.Unmarshal(ev.Payload, &n); err != nil {
			slog.ErrorContext(ctx, "malformed notification event", "err", err)
			return
		}
		writeSSE(c, "notification", newNotificationResponse(n))
	case message.EventType:
		var m domain.Message
		if err := json.Unmarshal(ev.Payload, &m); err != nil {
			slog.ErrorContext(ctx, "malformed message event", "err", err)
			return
		}
		writeSSE(c, "dm", newMessageResponse(m))
	default:
		slog.WarnContext(ctx, "unknown realtime event type", "type", ev.Type)
	}
}
