package discovery_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/discovery"
	"github.com/stretchr/testify/require"
)

func TestService_GetTrendingHashtags(t *testing.T) {
	ctx := context.Background()

	t.Run("success_24h", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetTrendingHashtagsLast24hFn: func(ctx context.Context, limit int32) ([]db.Hashtag, error) {
				return []db.Hashtag{{Text: "trending"}}, nil
			},
		}
		uc := discovery.New(store)
		res, err := uc.TrendingHashtags(ctx, 5)
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "trending", res[0].Text)
	})

	t.Run("fallback_alltime", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetTrendingHashtagsLast24hFn: func(ctx context.Context, limit int32) ([]db.Hashtag, error) {
				return []db.Hashtag{}, nil
			},
			GetTopHashtagsAllTimeFn: func(ctx context.Context, limit int32) ([]db.Hashtag, error) {
				return []db.Hashtag{{Text: "alltime"}}, nil
			},
		}
		uc := discovery.New(store)
		res, err := uc.TrendingHashtags(ctx, 5)
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "alltime", res[0].Text)
	})
}

func TestService_GetSuggestedUsers(t *testing.T) {
	ctx := context.Background()
	viewerID := int64(1)

	t.Run("personalized", func(t *testing.T) {
		store := &dbtest.MockStore{
			ListSuggestedUsersFn: func(ctx context.Context, arg db.ListSuggestedUsersParams) ([]db.ListSuggestedUsersRow, error) {
				return []db.ListSuggestedUsersRow{
					{User: db.User{ID: 100, Username: "suggested"}},
				}, nil
			},
		}
		uc := discovery.New(store)
		res, err := uc.SuggestedUsers(ctx, 0, 10, &viewerID)
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "suggested", res[0].Username)
	})

	t.Run("guest_fallback", func(t *testing.T) {
		store := &dbtest.MockStore{
			ListTopUsersFn: func(ctx context.Context, arg db.ListTopUsersParams) ([]db.User, error) {
				return []db.User{{ID: 200, Username: "top"}}, nil
			},
		}
		uc := discovery.New(store)
		res, err := uc.SuggestedUsers(ctx, 0, 10, nil)
		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "top", res[0].Username)
	})
}
