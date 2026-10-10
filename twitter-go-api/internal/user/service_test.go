package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/storage/storagetest"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type noopNotifier struct{}

func (noopNotifier) Dispatch(db.Notification) {}

func TestUpdateProfile_ClearingFields(t *testing.T) {
	ctx := context.Background()

	oldAvatar := storagetest.BaseURL + "/avatars/1/old.png"
	oldBanner := "https://lh3.googleusercontent.com/a/banner" // external, must not be deleted

	store := &dbtest.MockStore{
		GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
			return db.GetUserRow{
				User: db.User{
					ID:        arg.ID,
					AvatarURL: &oldAvatar,
					BannerURL: &oldBanner,
				},
			}, nil
		},
		UpdateUserProfileFn: func(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error) {
			if arg.AvatarURL != nil {
				t.Errorf("expected AvatarURL to be nil, got %v", *arg.AvatarURL)
			}
			if arg.BannerURL != nil {
				t.Errorf("expected BannerURL to be nil, got %v", *arg.BannerURL)
			}
			return db.User{ID: arg.ID}, nil
		},
	}

	storage := &storagetest.Fake{}

	uc := user.New(store, storage, nil)

	empty := ""
	input := user.UpdateProfileInput{
		AvatarKey: &empty,
		BannerKey: &empty,
	}

	_, err := uc.UpdateProfile(ctx, 1, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.Deleted) != 1 || storage.Deleted[0] != "avatars/1/old.png" {
		t.Fatalf("expected only the hosted avatar to be deleted, got %v", storage.Deleted)
	}
}

func TestUpdateProfile_RejectsForeignKeys(t *testing.T) {
	store := &dbtest.MockStore{
		GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
			return db.GetUserRow{User: db.User{ID: arg.ID}}, nil
		},
	}
	uc := user.New(store, &storagetest.Fake{}, nil)

	for _, input := range []user.UpdateProfileInput{
		{AvatarKey: dbtest.Ptr("avatars/2/x.png")},
		{AvatarKey: dbtest.Ptr("https://evil.example/pixel.png")},
		{BannerKey: dbtest.Ptr("avatars/1/x.png")},
	} {
		_, err := uc.UpdateProfile(context.Background(), 1, input)
		if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
			t.Fatalf("expected bad request for %+v, got %v", input, err)
		}
	}
}

func TestFollowUser(t *testing.T) {
	t.Run("cannot_follow_yourself", func(t *testing.T) {
		uc := user.New(nil, nil, nil)
		_, err := uc.Follow(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error for self-follow")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		store := &dbtest.MockStore{
			GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: 2}}, nil
			},
			FollowUserFn: func(ctx context.Context, arg db.FollowUserParams) (bool, error) {
				return true, nil
			},
		}
		uc := user.New(store, nil, noopNotifier{})
		inserted, err := uc.Follow(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inserted {
			t.Fatal("expected inserted to be true")
		}
	})
}

func TestUnfollowUser(t *testing.T) {
	ctx := context.Background()
	store := &dbtest.MockStore{
		GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
			return db.GetUserRow{User: db.User{ID: 2}}, nil
		},
		UnfollowUserFn: func(ctx context.Context, arg db.UnfollowUserParams) (bool, error) {
			return true, nil
		},
	}
	uc := user.New(store, nil, noopNotifier{})
	err := uc.Unfollow(ctx, 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFollowers_PagesByFollowTime(t *testing.T) {
	followedAt := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	store := &dbtest.MockStore{
		GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
			return db.GetUserRow{User: db.User{ID: 1}}, nil
		},
		ListFollowersFn: func(ctx context.Context, arg db.ListFollowersParams) ([]db.ListFollowersRow, error) {
			return []db.ListFollowersRow{
				{User: db.User{ID: 2, Username: "user2"}, IsFollowing: true, FollowedAt: followedAt},
				{User: db.User{ID: 3}},
			}, nil
		},
	}

	page, err := user.New(store, nil, noopNotifier{}).Followers(context.Background(), 1, pagination.Request{Limit: 1}, dbtest.Ptr[int64](1))

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "user2", page.Items[0].Username)
	require.True(t, page.Items[0].IsFollowing)
	require.Equal(t, &pagination.Cursor{Time: followedAt, ID: 2}, page.Next)
}

func TestCompleteOnboarding(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects_invalid_username", func(t *testing.T) {
		for _, name := range []string{"ab", "has space", "way_too_long_username_x", "émoji"} {
			_, err := user.New(&dbtest.MockStore{}, nil, nil).CompleteOnboarding(ctx, 1, name, "Alice")
			kind, _ := apperr.KindOf(err)
			require.Equal(t, apperr.KindBadRequest, kind, name)
		}
	})

	t.Run("reserved_username_is_a_conflict", func(t *testing.T) {
		for _, name := range []string{"explore", "Messages", "SEARCH"} {
			_, err := user.New(&dbtest.MockStore{}, nil, nil).CompleteOnboarding(ctx, 1, name, "Alice")
			kind, _ := apperr.KindOf(err)
			require.Equal(t, apperr.KindConflict, kind, name)
		}
	})

	t.Run("taken_username_is_a_conflict", func(t *testing.T) {
		store := &dbtest.MockStore{
			SetUsernameFn: func(ctx context.Context, arg db.SetUsernameParams) (db.User, error) {
				return db.User{}, &pgconn.PgError{Code: "23505"}
			},
		}
		_, err := user.New(store, nil, nil).CompleteOnboarding(ctx, 1, "alice", "Alice")
		kind, _ := apperr.KindOf(err)
		require.Equal(t, apperr.KindConflict, kind)
	})

	t.Run("success_marks_onboarded", func(t *testing.T) {
		now := time.Now()
		store := &dbtest.MockStore{
			SetUsernameFn: func(ctx context.Context, arg db.SetUsernameParams) (db.User, error) {
				return db.User{ID: 1, Username: arg.Username, DisplayName: arg.DisplayName, OnboardedAt: &now}, nil
			},
		}
		u, err := user.New(store, nil, nil).CompleteOnboarding(ctx, 1, "Alice_99", "  Alice  ")
		require.NoError(t, err)
		require.Equal(t, "Alice_99", u.Username)
		require.Equal(t, "Alice", *u.DisplayName)
		require.True(t, u.Onboarded)
	})
}
