package httpapi

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
)

// Omitted fields are left unchanged; "" clears a field.
type updateProfileRequest struct {
	DisplayName *string `json:"displayName" binding:"omitempty,max=50"`
	Bio         *string `json:"bio" binding:"omitempty,max=160"`
	Location    *string `json:"location" binding:"omitempty,max=30"`
	Website     *string `json:"website" binding:"omitempty,max=100,http_url"`
	AvatarKey   *string `json:"avatarKey" binding:"omitempty,max=200"`
	BannerKey   *string `json:"bannerKey" binding:"omitempty,max=200"`
}

type onboardingRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=20"`
	DisplayName string `json:"displayName" binding:"required,max=50"`
}

type usernameURIRequest struct {
	Username string `uri:"username" binding:"required,max=20"`
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
		Location:    req.Location,
		Website:     req.Website,
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
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}

	viewerID := optionalViewerID(c)

	u, err := s.svc.Users.Get(c.Request.Context(), path.ID, viewerID)
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
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}

	followerID, ok := requireUserID(c)
	if !ok {
		return
	}

	_, err := s.svc.Users.Follow(c.Request.Context(), followerID, path.ID)
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
	var path idURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}

	followerID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := s.svc.Users.Unfollow(c.Request.Context(), followerID, path.ID); err != nil {
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
	users, err := s.svc.Users.Followers(c.Request.Context(), path.ID, req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newPageResponse(users, newUserResponse))
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
	users, err := s.svc.Users.Following(c.Request.Context(), path.ID, req, viewerID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newPageResponse(users, newUserResponse))
}

// getUserByUsername godoc
// @Summary		Get User by Username
// @Description	Look a profile up by handle (case-insensitive).
// @Tags			Users
// @Produce		json
// @Param			username	path		string	true	"Username"
// @Success		200		{object}	UserResponse
// @Failure		404		{object}	ErrorResponse
// @Router			/users/by-username/{username} [get]
func (s *Server) getUserByUsername(c *gin.Context) {
	var path usernameURIRequest
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, err)
		return
	}
	u, err := s.svc.Users.GetByUsername(c.Request.Context(), path.Username, optionalViewerID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newUserResponse(u))
}

// completeOnboarding godoc
// @Summary		Complete Onboarding
// @Description	Choose a username (3-20 letters, numbers or underscores) and display name. Sign-up assigns a generated username until then.
// @Tags			Users
// @Accept			json
// @Produce		json
// @Param			request	body			onboardingRequest	true	"Chosen username and display name"
// @Success		200		{object}	CurrentUserResponse
// @Failure		400		{object}	ErrorResponse
// @Failure		409		{object}	ErrorResponse	"username taken"
// @Security		BearerAuth
// @Router			/users/me/onboarding [post]
func (s *Server) completeOnboarding(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req onboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}
	u, err := s.svc.Users.CompleteOnboarding(c.Request.Context(), userID, req.Username, req.DisplayName)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, newCurrentUserResponse(u))
}
