package httpapi

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
)

type updateProfileRequest struct {
	DisplayName *string `json:"displayName" binding:"omitempty,max=30"`
	Bio         *string `json:"bio" binding:"omitempty,max=160"`
	AvatarKey   *string `json:"avatarKey" binding:"omitempty"`
	BannerKey   *string `json:"bannerKey" binding:"omitempty"`
}

// updateProfile godoc
// @Summary		Update User Profile
// @Description	Modify the display name, bio, or profile images of the authenticated user.
// @Tags			Users
// @Accept			json
// @Produce		json
// @Param			request	body			updateProfileRequest	true	"Profile Update Details"
// @Success		200		{object}	UserResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		401		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/users/profile [put]
func (s *Server) updateProfile(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	input := user.UpdateProfileInput{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarKey:   req.AvatarKey,
		BannerKey:   req.BannerKey,
	}

	updatedUser, err := s.svc.Users.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newUserResponse(updatedUser))
}

// getUser godoc
// @Summary		Get User Profile
// @Description	Get public profile information for a user by their ID.
// @Tags			Users
// @Produce		json
// @Param			id	path		int64	true	"User ID"
// @Success		200	{object}	UserResponse
// @Failure		404	{object}	ErrorResponse
// @Router			/users/{id} [get]
func (s *Server) getUser(c *gin.Context) {
	var req idURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeError(c, err)
		return
	}

	viewerID := optionalViewerID(c)

	u, err := s.svc.Users.Get(c.Request.Context(), req.ID, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newUserResponse(u))
}

// followUser godoc
// @Summary		Follow User
// @Description	Establish a following relationship with another user.
// @Tags			Users
// @Produce		json
// @Param			id	path		int64	true	"User ID to follow"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/users/{id}/follow [post]
func (s *Server) followUser(c *gin.Context) {
	var req idURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeError(c, err)
		return
	}

	followerID, ok := requireUserID(c)
	if !ok {
		return
	}

	_, err := s.svc.Users.Follow(c.Request.Context(), followerID, req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// unfollowUser godoc
// @Summary		Unfollow User
// @Description	Remove an existing following relationship.
// @Tags			Users
// @Produce		json
// @Param			id	path		int64	true	"User ID to unfollow"
// @Success		200	{object}	SuccessResponse
// @Failure		401	{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/users/{id}/follow [delete]
func (s *Server) unfollowUser(c *gin.Context) {
	var req idURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		writeError(c, err)
		return
	}

	followerID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := s.svc.Users.Unfollow(c.Request.Context(), followerID, req.ID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// listFollowers godoc
// @Summary		List Followers
// @Description	Get a paginated list of users following the specified user.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[UserResponse]
// @Router			/users/{id}/followers [get]
func (s *Server) listFollowers(c *gin.Context) {
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
	users, err := s.svc.Users.Followers(c.Request.Context(), req.ID, page, size+1, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	response := newUserResponseList(users)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}

// listFollowing godoc
// @Summary		List Following
// @Description	Get a paginated list of users followed by the specified user.
// @Tags			Users
// @Produce		json
// @Param			id		path		int		true	"User ID"
// @Param			cursor	query		string	false	"Pagination cursor"
// @Param			size	query		int		false	"Number of items per page"
// @Success		200		{object}	PageResponse[UserResponse]
// @Router			/users/{id}/following [get]
func (s *Server) listFollowing(c *gin.Context) {
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
	users, err := s.svc.Users.Following(c.Request.Context(), req.ID, page, size+1, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	response := newUserResponseList(users)
	c.JSON(http.StatusOK, BuildPageResponse(response, size, offset))
}
