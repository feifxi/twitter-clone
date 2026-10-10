package feed_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/feed"
	"github.com/stretchr/testify/require"
)

func TestService_GetGlobalFeed(t *testing.T) {
	ctx := context.Background()
	viewerID := int64(1)

	t.Run("success", func(t *testing.T) {
		store := &dbtest.MockStore{
			ListForYouFeedFn: func(ctx context.Context, arg db.ListForYouFeedParams) ([]db.ListForYouFeedRow, error) {
				return []db.ListForYouFeedRow{
					{Tweet: db.Tweet{ID: 10, UserID: 100}, IsLiked: true},
				}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: 100, Username: "user100"}},
				}, nil
			},
		}

		uc := feed.New(store)
		feed, err := uc.Global(ctx, 0, 10, &viewerID)

		require.NoError(t, err)
		require.Len(t, feed, 1)
		require.Equal(t, int64(10), feed[0].ID)
		require.Equal(t, "user100", feed[0].Author.Username)
		require.True(t, feed[0].IsLiked)
	})
}

func TestService_GetFollowingFeed(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)

	t.Run("success", func(t *testing.T) {
		store := &dbtest.MockStore{
			ListFollowingFeedFn: func(ctx context.Context, arg db.ListFollowingFeedParams) ([]db.ListFollowingFeedRow, error) {
				return []db.ListFollowingFeedRow{
					{Tweet: db.Tweet{ID: 20, UserID: 200}, IsFollowing: true},
				}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: 200, Username: "user200"}},
				}, nil
			},
		}

		uc := feed.New(store)
		feed, err := uc.Following(ctx, userID, 0, 10)

		require.NoError(t, err)
		require.Len(t, feed, 1)
		require.Equal(t, int64(20), feed[0].ID)
		require.True(t, feed[0].IsFollowing)
	})
}

func TestService_GetUserFeed(t *testing.T) {
	ctx := context.Background()
	targetUserID := int64(100)

	t.Run("success", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: targetUserID}}, nil
			},
			ListUserTweetsFn: func(ctx context.Context, arg db.ListUserTweetsParams) ([]db.ListUserTweetsRow, error) {
				return []db.ListUserTweetsRow{
					{Tweet: db.Tweet{ID: 30, UserID: targetUserID}},
				}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: targetUserID, Username: "targetuser"}},
				}, nil
			},
		}

		uc := feed.New(store)
		feed, err := uc.ForUser(ctx, targetUserID, 0, 10, nil)

		require.NoError(t, err)
		require.Len(t, feed, 1)
		require.Equal(t, int64(30), feed[0].ID)
		require.Equal(t, "targetuser", feed[0].Author.Username)
	})
}
