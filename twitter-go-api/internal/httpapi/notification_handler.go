package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
