package api

import (
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-gonic/gin"
)

type googleLoginRequest struct {
	IdToken string `json:"idToken" binding:"required"`
}

// loginGoogle godoc
// @Summary		Login with Google
// @Description	Exchange a Google ID token for a session. The access and refresh tokens are set as httpOnly cookies.
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			request	body			googleLoginRequest	true	"Google ID Token"
// @Success		200		{object}	CurrentUserResponse
// @Failure		400		{object}	ErrorResponse
// @Router			/auth/google [post]
func (s *Server) loginGoogle(c *gin.Context) {
	var req googleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	session, err := s.svc.Auth.LoginWithGoogle(c.Request.Context(), req.IdToken)
	if err != nil {
		writeError(c, err)
		return
	}

	s.setSessionCookies(c, session)
	c.JSON(http.StatusOK, newCurrentUserResponse(session.User))
}

// refreshToken godoc
// @Summary		Refresh Access Token
// @Description	Rotate the session in the refresh_token cookie: both cookies are replaced and the old refresh token stops working.
// @Tags			Auth
// @Produce		json
// @Success		200		{object}	CurrentUserResponse
// @Failure		401		{object}	ErrorResponse
// @Router			/auth/refresh [post]
func (s *Server) refreshToken(c *gin.Context) {
	refreshToken, _ := c.Cookie(middleware.RefreshTokenCookie)
	if refreshToken == "" {
		writeError(c, apperr.Unauthorized("missing refresh token"))
		return
	}

	session, err := s.svc.Auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		// Cookies are deliberately left alone: when two tabs refresh at once, the
		// loser's 401 may arrive after the winner's response, and clearing here
		// would sign the user out of the session the winner just renewed.
		writeError(c, err)
		return
	}

	s.setSessionCookies(c, session)
	c.JSON(http.StatusOK, newCurrentUserResponse(session.User))
}

// logout godoc
// @Summary		Logout
// @Description	End this browser's session and clear its cookies. Other devices stay signed in.
// @Tags			Auth
// @Produce		json
// @Success		204		"No Content"
// @Router			/auth/logout [post]
func (s *Server) logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(middleware.RefreshTokenCookie)
	if err := s.svc.Auth.Logout(c.Request.Context(), refreshToken); err != nil {
		writeError(c, err)
		return
	}
	s.clearSessionCookies(c)
	c.Status(http.StatusNoContent)
}

// getMe godoc
// @Summary		Get Current User
// @Description	The signed-in user, with their email and onboarding state.
// @Tags			Users
// @Produce		json
// @Success		200		{object}	CurrentUserResponse
// @Failure		401		{object}	ErrorResponse
// @Security		BearerAuth
// @Router			/users/me [get]
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
