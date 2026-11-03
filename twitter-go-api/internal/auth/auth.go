// Package auth signs users in with Google and manages JWT access / refresh-token sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
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

// Session is a signed-in device: a short-lived access token plus the refresh
// token that renews it.
type Session struct {
	User             domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
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
	name = truncateRunes(strings.TrimSpace(name), maxDisplayNameLength)
	picture, _ := payload.Claims["picture"].(string)

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return Session{}, err
		}

		username := generatedUsername(email)
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

	return s.issueSession(ctx, domain.NewUser(user, false))
}

// Refresh rotates the session: the presented refresh token is consumed and a new
// pair is issued. A token works at most once; replaying it fails.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	// One DELETE ... RETURNING: two concurrent refreshes with the same token can't both succeed.
	session, err := s.store.ConsumeRefreshToken(ctx, hashRefreshToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, apperr.Unauthorized("refresh token not found or revoked")
	}
	if err != nil {
		return Session{}, err
	}
	if time.Now().After(session.ExpiryDate) {
		return Session{}, apperr.Unauthorized("refresh token expired")
	}

	row, err := s.store.GetUser(ctx, db.GetUserParams{ID: session.UserID})
	if err != nil {
		return Session{}, db.NotFound(err, "user")
	}

	return s.issueSession(ctx, domain.NewUser(row.User, false))
}

// Logout ends the session the refresh token belongs to; other devices stay signed in.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.store.DeleteRefreshToken(ctx, hashRefreshToken(refreshToken))
}

func (s *Service) Me(ctx context.Context, userID int64) (domain.User, error) {
	row, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID})
	if err != nil {
		return domain.User{}, db.NotFound(err, "user")
	}
	return domain.NewUser(row.User, row.IsFollowing), nil
}

// issueSession starts a new session (one per device) and prunes the user's expired ones.
func (s *Service) issueSession(ctx context.Context, user domain.User) (Session, error) {
	now := time.Now()
	accessToken, err := s.tokens.CreateToken(user.ID, s.cfg.AccessTokenTTL)
	if err != nil {
		return Session{}, err
	}

	refreshToken := rand.Text()
	refreshExpiresAt := now.Add(s.cfg.RefreshTokenTTL)
	if _, err := s.store.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:     user.ID,
		TokenHash:  hashRefreshToken(refreshToken),
		ExpiryDate: refreshExpiresAt,
	}); err != nil {
		return Session{}, err
	}
	if err := s.store.DeleteExpiredRefreshTokens(ctx, user.ID); err != nil {
		return Session{}, err
	}
	return Session{
		User:             user,
		AccessToken:      accessToken,
		AccessExpiresAt:  now.Add(s.cfg.AccessTokenTTL),
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

const (
	maxDisplayNameLength = 50
	usernameBaseLength   = 11 // + "_" + 8 random chars = 20, the column limit
)

// generatedUsername derives a placeholder handle from the email; users pick
// their own during onboarding.
func generatedUsername(email string) string {
	local, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	base := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, local)
	if base == "" {
		base = "user"
	}
	return truncateRunes(base, usernameBaseLength) + "_" + strings.ToLower(rand.Text()[:8])
}

func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
