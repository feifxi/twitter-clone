package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/idtoken"
)

type mockTokenMaker struct {
	token.Maker
	createTokenFn func(userID int64, duration time.Duration) (string, error)
}

func (m *mockTokenMaker) CreateToken(userID int64, duration time.Duration) (string, error) {
	return m.createTokenFn(userID, duration)
}

type mockGoogleVerifier struct {
	validateFn func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

func (m *mockGoogleVerifier) Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
	return m.validateFn(ctx, idToken, audience)
}

func TestService_LoginWithGoogle(t *testing.T) {
	cfg := auth.Config{
		GoogleClientID:  "test-client-id",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	t.Run("success_existing_user", func(t *testing.T) {
		ctx := context.Background()
		email := "test@example.com"
		userID := int64(123)

		gv := &mockGoogleVerifier{
			validateFn: func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
				return &idtoken.Payload{Claims: map[string]interface{}{"email": email, "email_verified": true}}, nil
			},
		}

		store := &dbtest.MockStore{
			GetUserByEmailFn: func(ctx context.Context, e string) (db.User, error) {
				return db.User{ID: userID, Email: e}, nil
			},
			DeleteRefreshTokensByUserFn: func(ctx context.Context, id int64) error {
				return nil
			},
			CreateRefreshTokenFn: func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
				return db.RefreshToken{}, nil
			},
		}

		tm := &mockTokenMaker{
			createTokenFn: func(userID int64, duration time.Duration) (string, error) {
				return "access-token", nil
			},
		}

		uc := auth.New(cfg, store, tm, gv)
		res, err := uc.LoginWithGoogle(ctx, "id-token")

		require.NoError(t, err)
		require.Equal(t, userID, res.User.ID)
		require.Equal(t, "access-token", res.AccessToken)
		require.NotEmpty(t, res.RefreshToken)
	})

	t.Run("success_new_user", func(t *testing.T) {
		ctx := context.Background()
		email := "new@example.com"
		userID := int64(456)

		gv := &mockGoogleVerifier{
			validateFn: func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
				return &idtoken.Payload{Claims: map[string]interface{}{"email": email, "email_verified": true, "name": "New User"}}, nil
			},
		}

		store := &dbtest.MockStore{
			GetUserByEmailFn: func(ctx context.Context, e string) (db.User, error) {
				return db.User{}, pgx.ErrNoRows
			},
			CreateUserFn: func(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
				return db.User{ID: userID, Email: email, Username: arg.Username}, nil
			},
			DeleteRefreshTokensByUserFn: func(ctx context.Context, id int64) error {
				return nil
			},
			CreateRefreshTokenFn: func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
				return db.RefreshToken{}, nil
			},
		}

		tm := &mockTokenMaker{
			createTokenFn: func(userID int64, duration time.Duration) (string, error) {
				return "access-token", nil
			},
		}

		uc := auth.New(cfg, store, tm, gv)
		res, err := uc.LoginWithGoogle(ctx, "id-token")

		require.NoError(t, err)
		require.Equal(t, userID, res.User.ID)
		require.NotEmpty(t, res.AccessToken)
		require.NotEmpty(t, res.RefreshToken)
	})
}

func TestService_LoginWithGoogle_RejectsUnverifiedEmail(t *testing.T) {
	gv := &mockGoogleVerifier{
		validateFn: func(context.Context, string, string) (*idtoken.Payload, error) {
			return &idtoken.Payload{Claims: map[string]interface{}{"email": "victim@example.com", "email_verified": false}}, nil
		},
	}
	uc := auth.New(auth.Config{}, &dbtest.MockStore{}, &mockTokenMaker{}, gv)

	_, err := uc.LoginWithGoogle(context.Background(), "id-token")

	kind, ok := apperr.KindOf(err)
	require.True(t, ok)
	require.Equal(t, apperr.KindUnauthorized, kind)
}

func TestService_RefreshSession(t *testing.T) {
	cfg := auth.Config{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(123)
		refreshToken := "valid-refresh-token"

		store := &dbtest.MockStore{
			GetRefreshTokenFn: func(ctx context.Context, hash string) (db.RefreshToken, error) {
				return db.RefreshToken{UserID: userID, ExpiryDate: time.Now().Add(time.Hour)}, nil
			},
			DeleteRefreshTokensByUserFn: func(ctx context.Context, id int64) error {
				return nil
			},
			CreateRefreshTokenFn: func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
				return db.RefreshToken{}, nil
			},
			GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: userID}}, nil
			},
		}

		tm := &mockTokenMaker{
			createTokenFn: func(userID int64, duration time.Duration) (string, error) {
				return "new-access-token", nil
			},
		}

		uc := auth.New(cfg, store, tm, nil)
		res, err := uc.Refresh(ctx, refreshToken)

		require.NoError(t, err)
		require.Equal(t, userID, res.User.ID)
		require.Equal(t, "new-access-token", res.AccessToken)
		require.NotEmpty(t, res.RefreshToken)
	})

	t.Run("expired_token", func(t *testing.T) {
		refreshToken := "expired-refresh-token"

		store := &dbtest.MockStore{
			GetRefreshTokenFn: func(ctx context.Context, hash string) (db.RefreshToken, error) {
				return db.RefreshToken{UserID: 123, ExpiryDate: time.Now().Add(-time.Hour)}, nil
			},
			DeleteRefreshTokenFn: func(ctx context.Context, hash string) error {
				return nil
			},
		}

		uc := auth.New(cfg, store, nil, nil)
		_, err := uc.Refresh(ctx, refreshToken)

		require.Error(t, err)
	})
}

func TestService_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("logout_by_userID", func(t *testing.T) {
		userID := int64(123)
		called := false
		store := &dbtest.MockStore{
			DeleteRefreshTokensByUserFn: func(ctx context.Context, id int64) error {
				called = true
				return nil
			},
		}

		uc := auth.New(auth.Config{}, store, nil, nil)
		uc.Logout(ctx, &userID, nil)
		require.True(t, called)
	})

	t.Run("logout_by_token", func(t *testing.T) {
		token := "some-token"
		called := false
		store := &dbtest.MockStore{
			DeleteRefreshTokenFn: func(ctx context.Context, hash string) error {
				called = true
				return nil
			},
		}

		uc := auth.New(auth.Config{}, store, nil, nil)
		uc.Logout(ctx, nil, &token)
		require.True(t, called)
	})
}
