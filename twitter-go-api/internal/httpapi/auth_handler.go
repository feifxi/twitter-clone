package httpapi

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
// @Description	Revoke the user session and clear access/refresh tokens.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body			refreshTokenRequest	false	"Optional Refresh Token to revoke explicitly"
// @Success		200		{object}	SuccessResponse
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/auth/logout [post]
func (s *Server) logout(c *gin.Context) {
	userID, ok := currentUserID(c)
	if ok {
		s.svc.Auth.Logout(c.Request.Context(), &userID, nil)
	} else if rt := resolveRefreshToken(c); rt != "" {
		s.svc.Auth.Logout(c.Request.Context(), nil, &rt)
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
