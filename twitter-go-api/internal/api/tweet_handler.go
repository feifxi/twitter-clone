package api

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/gin-gonic/gin"
)

type createTweetRequest struct {
	Content  *string `json:"content" binding:"required_without=MediaKey,omitempty,max=280"`
	ParentID *int64  `json:"parentId" binding:"omitempty,min=1"`
	MediaKey *string `json:"mediaKey" binding:"omitempty,max=200"`
}

// createTweet godoc
// @Summary		Create Tweet
// @Description	Post a new tweet with optional text and media.
// @Tags			Tweets
// @Accept			json
// @Produce		json
// @Param			request	body			createTweetRequest	true	"Tweet content and media"
// @Success		201		{object}	TweetResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets [post]
func (s *Server) createTweet(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req createTweetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	input := tweet.CreateInput{
		UserID:   userID,
		Content:  req.Content,
		ParentID: req.ParentID,
		MediaKey: req.MediaKey,
	}

	t, err := s.svc.Tweets.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, newTweetResponse(t))
}

// deleteTweet godoc
// @Summary		Delete Tweet
// @Description	Remove a tweet by ID. Only the author can delete their own tweet.
// @Tags			Tweets
// @Param			id	path		int64	true	"Tweet ID"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Failure		403	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets/{id} [delete]
func (s *Server) deleteTweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := s.svc.Tweets.Delete(c.Request.Context(), userID, path.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// getTweet godoc
// @Summary		Get Tweet
// @Description	Fetch a single tweet by its ID.
// @Tags			Tweets
// @Produce		json
// @Param			id	path		int64	true	"Tweet ID"
// @Success		200	{object}	TweetResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/tweets/{id} [get]
func (s *Server) getTweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	viewerID := optionalViewerID(c)
	t, err := s.svc.Tweets.Get(c.Request.Context(), path.ID, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newTweetResponse(t))
}

// getReplies godoc
// @Summary		Get Tweet Replies
// @Description	Get a paginated list of replies to a specific tweet.
// @Tags			Tweets
// @Produce		json
// @Param			id		path		int		true	"Tweet ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[TweetResponse]
// @Router			/tweets/{id}/replies [get]
func (s *Server) getReplies(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	req, ok := parsePageRequest(c)
	if !ok {
		return
	}
	viewerID := optionalViewerID(c)
	replies, err := s.svc.Tweets.ListReplies(c.Request.Context(), path.ID, req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newPageResponse(replies, newTweetResponse))
}

// likeTweet godoc
// @Summary		Like Tweet
// @Description	Add a like to a tweet.
// @Tags			Tweets
// @Param			id	path		int64	true	"Tweet ID"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets/{id}/like [post]
func (s *Server) likeTweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := s.svc.Tweets.Like(c.Request.Context(), userID, path.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// unlikeTweet godoc
// @Summary		Unlike Tweet
// @Description	Remove a like from a tweet.
// @Tags			Tweets
// @Param			id	path		int64	true	"Tweet ID"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets/{id}/like [delete]
func (s *Server) unlikeTweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := s.svc.Tweets.Unlike(c.Request.Context(), userID, path.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// retweet godoc
// @Summary		Retweet
// @Description	Share a tweet on your own profile.
// @Tags			Tweets
// @Produce		json
// @Param			id	path		int64	true	"Tweet ID to retweet"
// @Success		200	{object}	TweetResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets/{id}/retweet [post]
func (s *Server) retweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	t, err := s.svc.Tweets.Retweet(c.Request.Context(), userID, path.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newTweetResponse(t))
}

// undoRetweet godoc
// @Summary		Undo Retweet
// @Description	Remove a previously shared retweet.
// @Tags			Tweets
// @Param			id	path		int64	true	"Original Tweet ID"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/tweets/{id}/retweet [delete]
func (s *Server) undoRetweet(c *gin.Context) {
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := s.svc.Tweets.UndoRetweet(c.Request.Context(), userID, path.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}
