package api

import (
	"strconv"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
)

// PageResponse is the envelope of every paginated list. nextCursor is opaque:
// clients pass it back as ?cursor= and must not interpret it.
type PageResponse[T any] struct {
	Items      []T     `json:"items"`
	HasNext    bool    `json:"hasNext"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

func newPageResponse[T, R any](page pagination.Page[T], toResponse func(T) R) PageResponse[R] {
	items := make([]R, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toResponse(item))
	}
	resp := PageResponse[R]{Items: items, HasNext: page.Next != nil}
	if page.Next != nil {
		cursor := pagination.Encode(*page.Next)
		resp.NextCursor = &cursor
	}
	return resp
}

type pageQuery struct {
	Cursor string `form:"cursor"`
	Size   *int32 `form:"size" binding:"omitempty,min=1"`
}

// parsePageRequest reads ?cursor=&size= (default 20, capped at 50).
func parsePageRequest(c *gin.Context) (pagination.Request, bool) {
	var q pageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		writeError(c, err)
		return pagination.Request{}, false
	}

	req := pagination.Request{Limit: pagination.DefaultLimit}
	if q.Size != nil {
		req.Limit = min(*q.Size, pagination.MaxLimit)
	}
	if q.Cursor != "" {
		after, err := pagination.Decode(q.Cursor)
		if err != nil {
			writeValidationError(c, "cursor", "invalid cursor")
			return pagination.Request{}, false
		}
		req.After = after
	}
	return req, true
}

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

// parseLimit reads a "limit" query param for small, unpaginated lists.
func parseLimit(c *gin.Context, defaultVal, max int32) int32 {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed < 1 {
		return defaultVal
	}
	return min(int32(parsed), max)
}
