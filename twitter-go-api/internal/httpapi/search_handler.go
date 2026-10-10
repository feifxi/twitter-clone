package httpapi

import (
	"net/http"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/gin-gonic/gin"
)

// searchUsers godoc
// @Summary		Search Users
// @Description	Search for users by username or display name with pagination.
// @Tags			Search
// @Produce		json
// @Param			q		query		string	true	"Search query"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[UserResponse]
// @Router			/search/users [get]
func (s *Server) searchUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeError(c, apperr.BadRequest("search query is required"))
		return
	}

	req, ok := parsePageRequest(c)
	if !ok {
		return
	}

	viewerID := optionalViewerID(c)
	users, err := s.svc.Search.Users(c.Request.Context(), query, req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newPageResponse(users, newUserResponse))
}

// searchTweets godoc
// @Summary		Search Tweets
// @Description	Search for tweets by content (using full-text search) with pagination.
// @Tags			Search
// @Produce		json
// @Param			q		query		string	true	"Search query"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[TweetResponse]
// @Router			/search/tweets [get]
func (s *Server) searchTweets(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeError(c, apperr.BadRequest("search query is required"))
		return
	}

	req, ok := parsePageRequest(c)
	if !ok {
		return
	}

	viewerID := optionalViewerID(c)
	tweets, err := s.svc.Search.Tweets(c.Request.Context(), query, req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newPageResponse(tweets, newTweetResponse))
}

// searchHashtags godoc
// @Summary		Search Hashtags
// @Description	Find trending or specific hashtags matching a prefix.
// @Tags			Search
// @Produce		json
// @Param			q		query		string	true	"Hashtag prefix"
// @Param			limit	query		int		false	"Max number of results"
// @Success		200		{array}		HashtagResponse
// @Router			/search/hashtags [get]
func (s *Server) searchHashtags(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	limit := parseLimit(c, 5, pagination.MaxLimit)
	hashtags, err := s.svc.Search.Hashtags(c.Request.Context(), query, limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newHashtagResponseList(hashtags)
	c.JSON(http.StatusOK, response)
}
