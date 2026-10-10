package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/idtoken"
)

type AuthResult struct {
	User         UserItem
	AccessToken  string
	RefreshToken string
}

type realGoogleVerifier struct{}

func (v *realGoogleVerifier) Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, idToken, audience)
}

func NewRealGoogleVerifier() GoogleVerifier {
	return &realGoogleVerifier{}
}

func (u *AuthUsecase) LoginWithGoogle(ctx context.Context, idToken string) (AuthResult, error) {
	payload, err := u.googleVerifier.Validate(ctx, idToken, u.config.GoogleClientID)
	if err != nil {
		return AuthResult{}, apperr.Unauthorized("invalid google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return AuthResult{}, apperr.Unauthorized("invalid google token")
	}
	// Accounts are keyed by email, so an unverified email would let someone claim another person's account.
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		return AuthResult{}, apperr.Unauthorized("google email is not verified")
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	user, err := u.store.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return AuthResult{}, err
		}

		username := buildUniqueUsername(email)
		user, err = u.store.CreateUser(ctx, db.CreateUserParams{
			Username:    username,
			Email:       email,
			DisplayName: &name,
			AvatarUrl:   &picture,
			BannerUrl:   nil,
			Role:        RoleUser,
			Provider:    "GOOGLE",
		})
		if err != nil {
			return AuthResult{}, err
		}
	}

	accessToken, refreshToken, err := u.issueSession(ctx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{User: newUserItemFromDB(user, false), AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (u *AuthUsecase) RefreshSession(ctx context.Context, refreshToken string) (AuthResult, error) {
	tokenHash := hashRefreshToken(refreshToken)
	session, err := u.store.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return AuthResult{}, apperr.Unauthorized("refresh token not found or revoked")
	}

	if time.Now().After(session.ExpiryDate) {
		_ = u.store.DeleteRefreshToken(ctx, tokenHash)
		return AuthResult{}, apperr.Unauthorized("refresh token expired")
	}

	accessToken, newRefreshToken, err := u.issueSession(ctx, session.UserID)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := u.store.GetUser(ctx, db.GetUserParams{ID: session.UserID, ViewerID: nil})
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:         newUserItemFromDB(user.User, user.IsFollowing),
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (u *AuthUsecase) Logout(ctx context.Context, userID *int64, refreshToken *string) {
	if userID != nil {
		_ = u.store.DeleteRefreshTokensByUser(ctx, *userID)
		return
	}
	if refreshToken != nil && strings.TrimSpace(*refreshToken) != "" {
		_ = u.store.DeleteRefreshToken(ctx, hashRefreshToken(*refreshToken))
	}
}

func (u *AuthUsecase) GetMe(ctx context.Context, userID int64) (UserItem, error) {
	user, err := u.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: nil})
	if err != nil {
		return UserItem{}, err
	}
	return newUserItemFromDB(user.User, user.IsFollowing), nil
}

func (u *AuthUsecase) issueSession(ctx context.Context, userID int64) (accessToken string, refreshToken string, err error) {
	accessToken, err = u.tokenMaker.CreateToken(userID, time.Duration(u.config.TokenDurationMinutes)*time.Minute)
	if err != nil {
		return "", "", err
	}

	refreshToken = uuid.NewString()
	expiresAt := time.Now().Add(time.Duration(u.config.RefreshTokenDurationDays) * 24 * time.Hour)

	if err := u.store.DeleteRefreshTokensByUser(ctx, userID); err != nil {
		return "", "", err
	}

	_, err = u.store.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:     userID,
		TokenHash:  hashRefreshToken(refreshToken),
		ExpiryDate: expiresAt,
	})
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func buildUniqueUsername(email string) string {
	base := strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")[0]
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, base)
	if clean == "" {
		clean = "user"
	}
	return fmt.Sprintf("%s_%s", clean, strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
