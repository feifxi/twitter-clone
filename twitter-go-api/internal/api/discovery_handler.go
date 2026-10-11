package api

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/gin-gonic/gin"
)

// getTrendingHashtags godoc
// @Summary		Trending Hashtags
// @Description	Get a list of currently trending hashtags based on recent usage.
// @Tags			Discovery
// @Produce		json
// @Param			limit	query		int		false	"Max number of results"
// @Success		200		{array}		HashtagResponse
// @Router			/discovery/trending [get]
func (s *Server) getTrendingHashtags(c *gin.Context) {
	limit := parseLimit(c, 10, pagination.MaxLimit)
	hashtags, err := s.svc.Discovery.TrendingHashtags(c.Request.Context(), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newHashtagResponseList(hashtags)
	c.JSON(http.StatusOK, response)
}

// getSuggestedUsers godoc
// @Summary		Suggested Users
// @Description	Get a paginated list of suggested users to follow.
// @Tags			Discovery
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[UserResponse]
// @Router			/discovery/users [get]
func (s *Server) getSuggestedUsers(c *gin.Context) {
	req, ok := parsePageRequest(c)
	if !ok {
		return
	}
	viewerID := optionalViewerID(c)
	users, err := s.svc.Discovery.SuggestedUsers(c.Request.Context(), req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newPageResponse(users, newUserResponse))
}
