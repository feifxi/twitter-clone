package httpapi

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
)

const (
	defaultSize = int32(20)
	maxSize     = int32(50)
)

type PageResponse[T any] struct {
	Items      []T     `json:"items"`
	HasNext    bool    `json:"hasNext"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

func BuildPageResponse[T any](items []T, size, offset int32) PageResponse[T] {
	if int32(len(items)) > size {
		next := encodeCursor(offset + size)
		return PageResponse[T]{
			Items:      items[:size],
			HasNext:    true,
			NextCursor: &next,
		}
	}
	return PageResponse[T]{
		Items:   items,
		HasNext: false,
	}
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

func requireUserID(c *gin.Context) (int64, bool) {
	userID, ok := currentUserID(c)
	if !ok {
		writeError(c, apperr.Unauthorized("authentication required"))
		return 0, false
	}
	return userID, true
}

// optionalViewerID returns a pointer to the current user's ID if authenticated, or nil otherwise.
func optionalViewerID(c *gin.Context) *int64 {
	if id, ok := currentUserID(c); ok {
		return &id
	}
	return nil
}

// parseLimit reads a "limit" query param, applying a default and max cap.
func parseLimit(c *gin.Context, defaultVal, max int32) int32 {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed < 1 {
		return defaultVal
	}
	limit := int32(parsed)
	if limit > max {
		return max
	}
	return limit
}

func parseOffsetAndSize(c *gin.Context) (int32, int32, bool) {
	type paginationQuery struct {
		Cursor *string `form:"cursor" binding:"omitempty"`
		Size   *int32  `form:"size" binding:"omitempty,min=1"`
	}

	var req paginationQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, err)
		return 0, 0, false
	}

	size := defaultSize
	if req.Size != nil {
		size = min(*req.Size, maxSize)
	}

	if req.Cursor == nil || *req.Cursor == "" {
		return 0, size, true
	}

	offset, err := decodeCursor(*req.Cursor)
	if err != nil || offset < 0 {
		writeValidationError(c, "cursor", "invalid cursor")
		return 0, 0, false
	}
	return offset, size, true
}

func encodeCursor(offset int32) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(int64(offset), 10)))
}

func decodeCursor(cursor string) (int32, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(string(decoded), 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}
