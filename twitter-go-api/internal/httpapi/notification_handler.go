package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/realtime"
	"github.com/gin-gonic/gin"
)

const sseHeartbeatInterval = 15 * time.Second

// streamNotifications godoc
// @Summary		Stream Real-time Notifications (SSE)
// @Description	Establish a Server-Sent Events (SSE) connection to receive near real-time notifications for likes, replies, retweets, and new followers.
// @Tags			Notifications
// @Produce			text/event-stream
// @Success		200	{string}	string	"text/event-stream"
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/notifications/stream [get]
func (s *Server) streamNotifications(c *gin.Context) {
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
			s.writeRealtimeEvent(c, ev)
		}
	}
}

// writeRealtimeEvent maps hub payloads (domain types) to the public DTOs before writing.
func (s *Server) writeRealtimeEvent(c *gin.Context, ev realtime.Event) {
	switch ev.Type {
	case notification.EventType:
		var n domain.Notification
		if err := json.Unmarshal(ev.Payload, &n); err != nil {
			slog.ErrorContext(c.Request.Context(), "malformed notification event", "err", err)
			return
		}
		writeSSE(c, "notification", newNotificationResponse(n))
	default:
		slog.WarnContext(c.Request.Context(), "unknown realtime event type", "type", ev.Type)
	}
}

// listNotifications godoc
// @Summary		List Notifications
// @Description	Get a paginated list of social notifications (likes, follows, etc.) for the current user.
// @Tags			Notifications
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[NotificationResponse]
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/notifications [get]
func (s *Server) listNotifications(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	items, err := s.svc.Notifications.List(c.Request.Context(), userID, offset/size, size+1)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, BuildPageResponse(newNotificationResponseList(items), size, offset))
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// getUnreadNotificationCount godoc
// @Summary		Get Unread Count
// @Description	Calculate how many unread notifications the current user has.
// @Tags			Notifications
// @Produce		json
// @Success		200	{object}	UnreadCountResponse
// @Security		BearerAuth
// @Failure		401	{object}	ErrorResponse
// @Router			/notifications/unread-count [get]
func (s *Server) getUnreadNotificationCount(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	count, err := s.svc.Notifications.CountUnread(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, UnreadCountResponse{Count: count})
}

// markNotificationRead godoc
// @Summary		Mark All Read
// @Description	Clear all unread notification badges for the current user.
// @Tags			Notifications
// @Success		200	{object}	SuccessResponse
// @Security		BearerAuth
// @Failure		401	{object}	ErrorResponse
// @Router			/notifications/mark-read [post]
func (s *Server) markNotificationRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := s.svc.Notifications.MarkAllRead(c.Request.Context(), userID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}
