package api

import (
	"context"
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/gin-gonic/gin"
)

// getGlobalFeed godoc
// @Summary		Global Feed
// @Description	Get a paginated list of all tweets.
// @Tags			Feeds
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Router			/feeds/global [get]
func (s *Server) getGlobalFeed(c *gin.Context) {
	req, ok := parsePageRequest(c)
	if !ok {
		return
	}
	viewerID := optionalViewerID(c)
	tweets, err := s.svc.Feeds.Global(c.Request.Context(), req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newPageResponse(tweets, newFeedItemResponse))
}

// getFollowingFeed godoc
// @Summary		Following Feed
// @Description	Get a paginated list of tweets from users the current user follows.
// @Tags			Feeds
// @Produce		json
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/feeds/following [get]
func (s *Server) getFollowingFeed(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	req, ok := parsePageRequest(c)
	if !ok {
		return
	}
	tweets, err := s.svc.Feeds.Following(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newPageResponse(tweets, newFeedItemResponse))
}

// The profile tabs share one shape: GET /users/{id}/<tab>?cursor=&size=.
type profileTab func(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.FeedItem], error)

func (s *Server) serveProfileTab(c *gin.Context, tab profileTab) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	req, ok := parsePageRequest(c)
	if !ok {
		return
	}
	page, err := tab(c.Request.Context(), path.ID, req, optionalViewerID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newPageResponse(page, newFeedItemResponse))
}

// getUserPosts godoc
// @Summary		Profile: Posts
// @Description	A user's top-level tweets and retweets, newest first.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Router			/users/{id}/posts [get]
func (s *Server) getUserPosts(c *gin.Context) { s.serveProfileTab(c, s.svc.Feeds.Posts) }

// getUserReplies godoc
// @Summary		Profile: Replies
// @Description	Replies written by a user, newest first.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Router			/users/{id}/replies [get]
func (s *Server) getUserReplies(c *gin.Context) { s.serveProfileTab(c, s.svc.Feeds.Replies) }

// getUserMedia godoc
// @Summary		Profile: Media
// @Description	A user's tweets that carry an image or video, newest first.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Router			/users/{id}/media [get]
func (s *Server) getUserMedia(c *gin.Context) { s.serveProfileTab(c, s.svc.Feeds.Media) }

// getUserLikes godoc
// @Summary		Profile: Likes
// @Description	Tweets the user liked, most recently liked first. Only the user themselves may list their likes.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[FeedItemResponse]
// @Failure		403		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/users/{id}/likes [get]
func (s *Server) getUserLikes(c *gin.Context) { s.serveProfileTab(c, s.svc.Feeds.Likes) }
