// Package auth signs users in with Google and manages JWT access / refresh-token sessions.
package auth

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
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/idtoken"
)

type Config struct {
	GoogleClientID  string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type tokenMaker interface {
	CreateToken(userID int64, duration time.Duration) (string, error)
}

// GoogleVerifier validates Google ID tokens; swapped for a fake in tests.
type GoogleVerifier interface {
	Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

// GoogleIDTokenVerifier verifies tokens against Google's public keys.
type GoogleIDTokenVerifier struct{}

func (GoogleIDTokenVerifier) Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, idToken, audience)
}

type Service struct {
	cfg      Config
	store    db.Store
	tokens   tokenMaker
	verifier GoogleVerifier
}

func New(cfg Config, store db.Store, tokens tokenMaker, verifier GoogleVerifier) *Service {
	return &Service{cfg: cfg, store: store, tokens: tokens, verifier: verifier}
}

type Session struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
}

func (s *Service) LoginWithGoogle(ctx context.Context, idToken string) (Session, error) {
	payload, err := s.verifier.Validate(ctx, idToken, s.cfg.GoogleClientID)
	if err != nil {
		return Session{}, apperr.Unauthorized("invalid google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return Session{}, apperr.Unauthorized("invalid google token")
	}
	// Accounts are keyed by email, so an unverified email would let someone claim another person's account.
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		return Session{}, apperr.Unauthorized("google email is not verified")
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return Session{}, err
		}

		username := buildUniqueUsername(email)
		user, err = s.store.CreateUser(ctx, db.CreateUserParams{
			Username:    username,
			Email:       email,
			DisplayName: &name,
			AvatarURL:   &picture,
			BannerURL:   nil,
			Role:        domain.RoleUser,
			Provider:    "GOOGLE",
		})
		if err != nil {
			return Session{}, err
		}
	}

	accessToken, refreshToken, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}

	return Session{User: domain.NewUser(user, false), AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	tokenHash := hashRefreshToken(refreshToken)
	session, err := s.store.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return Session{}, apperr.Unauthorized("refresh token not found or revoked")
	}

	if time.Now().After(session.ExpiryDate) {
		_ = s.store.DeleteRefreshToken(ctx, tokenHash)
		return Session{}, apperr.Unauthorized("refresh token expired")
	}

	accessToken, newRefreshToken, err := s.issueSession(ctx, session.UserID)
	if err != nil {
		return Session{}, err
	}

	user, err := s.store.GetUser(ctx, db.GetUserParams{ID: session.UserID, ViewerID: nil})
	if err != nil {
		return Session{}, err
	}

	return Session{
		User:         domain.NewUser(user.User, user.IsFollowing),
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID *int64, refreshToken *string) {
	if userID != nil {
		_ = s.store.DeleteRefreshTokensByUser(ctx, *userID)
		return
	}
	if refreshToken != nil && strings.TrimSpace(*refreshToken) != "" {
		_ = s.store.DeleteRefreshToken(ctx, hashRefreshToken(*refreshToken))
	}
}

func (s *Service) Me(ctx context.Context, userID int64) (domain.User, error) {
	user, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: nil})
	if err != nil {
		return domain.User{}, err
	}
	return domain.NewUser(user.User, user.IsFollowing), nil
}

func (s *Service) issueSession(ctx context.Context, userID int64) (string, string, error) {
	accessToken, err := s.tokens.CreateToken(userID, s.cfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.NewString()
	expiresAt := time.Now().Add(s.cfg.RefreshTokenTTL)

	if err := s.store.DeleteRefreshTokensByUser(ctx, userID); err != nil {
		return "", "", err
	}

	_, err = s.store.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
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
