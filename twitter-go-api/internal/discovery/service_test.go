package discovery_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/discovery"
	"github.com/chanombude/twitter-go-api/internal/pagination"
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

func TestService_SuggestedUsers(t *testing.T) {
	store := &dbtest.MockStore{
		ListSuggestedUsersFn: func(ctx context.Context, arg db.ListSuggestedUsersParams) ([]db.ListSuggestedUsersRow, error) {
			return []db.ListSuggestedUsersRow{{User: db.User{ID: 2, Username: "suggested"}}, {User: db.User{ID: 3}}}, nil
		},
	}

	page, err := discovery.New(store).SuggestedUsers(context.Background(), pagination.Request{Limit: 1}, dbtest.Ptr(int64(1)))

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "suggested", page.Items[0].Username)
	require.Equal(t, int32(1), page.Next.Offset)
}
