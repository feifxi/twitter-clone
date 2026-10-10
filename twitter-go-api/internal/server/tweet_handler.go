package server

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/usecase"
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
func (server *Server) createTweet(ctx *gin.Context) {
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}

	var req createTweetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, err)
		return
	}

	input := usecase.CreateTweetInput{
		UserID:   userID,
		Content:  req.Content,
		ParentID: req.ParentID,
		MediaKey: req.MediaKey,
	}

	tweet, err := server.tweetUC.CreateTweet(ctx, input)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, newTweetResponse(tweet))
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
func (server *Server) deleteTweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}
	if err := server.tweetUC.DeleteTweet(ctx, userID, req.ID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, successResponse())
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
func (server *Server) getTweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	viewerID := optionalViewerID(ctx)
	tweet, err := server.tweetUC.GetTweet(ctx, req.ID, viewerID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newTweetResponse(tweet))
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
func (server *Server) getReplies(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	offset, size, ok := parseOffsetAndSize(ctx)
	if !ok {
		return
	}
	page := offset / size
	viewerID := optionalViewerID(ctx)
	replies, err := server.tweetUC.ListReplies(ctx, req.ID, page, size+1, viewerID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	response := newTweetResponseList(replies)
	ctx.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
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
func (server *Server) likeTweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}
	if err := server.tweetUC.LikeTweet(ctx, userID, req.ID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, successResponse())
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
func (server *Server) unlikeTweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}
	if err := server.tweetUC.UnlikeTweet(ctx, userID, req.ID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, successResponse())
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
func (server *Server) retweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}
	tweet, err := server.tweetUC.Retweet(ctx, userID, req.ID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newTweetResponse(tweet))
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
func (server *Server) undoRetweet(ctx *gin.Context) {
	var req idURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		writeError(ctx, err)
		return
	}
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}
	if err := server.tweetUC.UndoRetweet(ctx, userID, req.ID); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, successResponse())
}
