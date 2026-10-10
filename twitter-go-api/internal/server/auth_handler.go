package server

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
func (server *Server) loginGoogle(ctx *gin.Context) {
	var req googleLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, err)
		return
	}

	authData, err := server.authUC.LoginWithGoogle(ctx, req.IdToken)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newAuthResponse(authData.AccessToken, authData.RefreshToken, authData.User))
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
func (server *Server) refreshToken(ctx *gin.Context) {
	refreshToken := resolveRefreshToken(ctx)
	if refreshToken == "" {
		writeError(ctx, apperr.Unauthorized("missing refresh token"))
		return
	}

	authData, err := server.authUC.RefreshSession(ctx, refreshToken)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newAuthResponse(authData.AccessToken, authData.RefreshToken, authData.User))
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
func (server *Server) logout(ctx *gin.Context) {
	userID, ok := getCurrentUserID(ctx)
	if ok {
		server.authUC.Logout(ctx, &userID, nil)
	} else if rt := resolveRefreshToken(ctx); rt != "" {
		server.authUC.Logout(ctx, nil, &rt)
	}

	ctx.JSON(http.StatusOK, successResponse())
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
func (server *Server) getMe(ctx *gin.Context) {
	userID, ok := mustCurrentUserID(ctx)
	if !ok {
		return
	}

	user, err := server.authUC.GetMe(ctx, userID)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newCurrentUserResponse(user))
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// resolveRefreshToken reads the refresh token from the JSON body field.
func resolveRefreshToken(ctx *gin.Context) string {
	var body refreshTokenRequest
	if ctx.ShouldBindJSON(&body) == nil && strings.TrimSpace(body.RefreshToken) != "" {
		return body.RefreshToken
	}
	return ""
}