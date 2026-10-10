package api

import (
	"net/http"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

type googleLoginRequest struct {
	IdToken string `json:"idToken" binding:"required"`
}

// loginGoogle godoc
// @Summary		Login with Google
// @Description	Exchange a Google ID token for a JWT access and refresh token.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body			googleLoginRequest	true	"Google ID Token"
// @Success		200		{object}	AuthResponse
// @Failure		400		{object}	ErrorResponse
// @Router			/auth/google [post]
func (s *Server) loginGoogle(c *gin.Context) {
	var req googleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	authData, err := s.svc.Auth.LoginWithGoogle(c.Request.Context(), req.IdToken)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newAuthResponse(authData.AccessToken, authData.RefreshToken, authData.User))
}

// refreshToken godoc
// @Summary		Refresh Access Token
// @Description	Obtain a new access token and refresh token using an existing refresh token.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body			refreshTokenRequest	true	"Refresh Token"
// @Success		200		{object}	AuthResponse
// @Failure		401		{object}	ErrorResponse
// @Router			/auth/refresh [post]
func (s *Server) refreshToken(c *gin.Context) {
	refreshToken := resolveRefreshToken(c)
	if refreshToken == "" {
		writeError(c, apperr.Unauthorized("missing refresh token"))
		return
	}

	authData, err := s.svc.Auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newAuthResponse(authData.AccessToken, authData.RefreshToken, authData.User))
}

// logout godoc
// @Summary		Logout
// @Description	End the session the refresh token belongs to. Other devices stay signed in.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body			refreshTokenRequest	true	"Refresh token of the session to end"
// @Success		200		{object}	SuccessResponse
// @Router			/auth/logout [post]
func (s *Server) logout(c *gin.Context) {
	if err := s.svc.Auth.Logout(c.Request.Context(), resolveRefreshToken(c)); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResponse())
}

// getMe godoc
// @Summary		Get Current User
// @Description	Fetch the profile of the currently authenticated user.
// @Tags			Auth
// @Produce		json
// @Success		200		{object}	CurrentUserResponse
// @Failure		401		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/auth/me [get]
func (s *Server) getMe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	user, err := s.svc.Auth.Me(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newCurrentUserResponse(user))
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// resolveRefreshToken reads the refresh token from the JSON body field.
func resolveRefreshToken(c *gin.Context) string {
	var body refreshTokenRequest
	if c.ShouldBindJSON(&body) == nil && strings.TrimSpace(body.RefreshToken) != "" {
		return body.RefreshToken
	}
	return ""
}
