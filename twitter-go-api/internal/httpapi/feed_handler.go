package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getGlobalFeed godoc
// @Summary		Global Feed
// @Description	Get a paginated list of all tweets.
// @Tags			Feeds
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[TweetResponse]
// @Router			/feeds/global [get]
func (s *Server) getGlobalFeed(c *gin.Context) {
	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	page := offset / size
	viewerID := optionalViewerID(c)
	tweets, err := s.svc.Feeds.Global(c.Request.Context(), page, size+1, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newTweetResponseList(tweets)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}

// getFollowingFeed godoc
// @Summary		Following Feed
// @Description	Get a paginated list of tweets from users the current user follows.
// @Tags			Feeds
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[TweetResponse]
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/feeds/following [get]
func (s *Server) getFollowingFeed(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	page := offset / size
	tweets, err := s.svc.Feeds.Following(c.Request.Context(), userID, page, size+1)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newTweetResponseList(tweets)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}

// getUserFeed godoc
// @Summary		User Profile Feed
// @Description	Get a paginated list of tweets for a specific user profile.
// @Tags			Feeds
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[TweetResponse]
// @Router			/feeds/user/{id} [get]
func (s *Server) getUserFeed(c *gin.Context) {
	var req idURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeError(c, err)
		return
	}
	offset, size, ok := parseOffsetAndSize(c)
	if !ok {
		return
	}
	page := offset / size
	viewerID := optionalViewerID(c)
	tweets, err := s.svc.Feeds.ForUser(c.Request.Context(), req.ID, page, size+1, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	response := newTweetResponseList(tweets)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}
