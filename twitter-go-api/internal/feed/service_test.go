package feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/feed"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/stretchr/testify/require"
)

func usersStore(store *dbtest.MockStore, users ...db.User) *dbtest.MockStore {
	store.GetUsersByIDsFn = func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
		rows := make([]db.GetUsersByIDsRow, 0, len(users))
		for _, u := range users {
			rows = append(rows, db.GetUsersByIDsRow{User: u})
		}
		return rows, nil
	}
	return store
}

func TestGlobal_KeepsTheAnchorAcrossPages(t *testing.T) {
	ctx := context.Background()
	anchor := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	var got db.ListForYouFeedParams

	store := usersStore(&dbtest.MockStore{
		ListForYouFeedFn: func(ctx context.Context, arg db.ListForYouFeedParams) ([]db.ListForYouFeedRow, error) {
			got = arg
			return []db.ListForYouFeedRow{
				{Tweet: db.Tweet{ID: 10, UserID: 1}, Score: 3.5},
				{Tweet: db.Tweet{ID: 9, UserID: 1}, Score: 2.5},
			}, nil
		},
	}, db.User{ID: 1, Username: "alice"})

	page, err := feed.New(store).Global(ctx, pagination.Request{After: &pagination.Cursor{Anchor: anchor, Score: 4, ID: 11}, Limit: 1}, nil)

	require.NoError(t, err)
	require.Equal(t, anchor, got.Anchor, "later pages must score against the first page's anchor")
	require.Equal(t, 4.0, *got.CursorScore)
	require.Equal(t, int32(2), got.PageLimit, "fetches one extra row to detect the next page")
	require.Len(t, page.Items, 1)
	require.Equal(t, "alice", page.Items[0].Tweet.Author.Username)
	require.Equal(t, &pagination.Cursor{Anchor: anchor, Score: 3.5, ID: 10}, page.Next)
}

func TestFollowing_HydratesRetweetsAndBuildsTimelineCursor(t *testing.T) {
	ctx := context.Background()
	retweetedAt := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	store := usersStore(&dbtest.MockStore{
		ListFollowingTimelineFn: func(ctx context.Context, arg db.ListFollowingTimelineParams) ([]db.ListFollowingTimelineRow, error) {
			return []db.ListFollowingTimelineRow{
				{Tweet: db.Tweet{ID: 20, UserID: 2}, RetweetID: dbtest.Ptr(int64(7)), RetweetedByID: dbtest.Ptr(int64(3)), RetweetedAt: &retweetedAt},
				{Tweet: db.Tweet{ID: 19, UserID: 2}},
			}, nil
		},
	}, db.User{ID: 2, Username: "author"}, db.User{ID: 3, Username: "retweeter"})

	page, err := feed.New(store).Following(ctx, 1, pagination.Request{Limit: 1})

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "author", page.Items[0].Tweet.Author.Username)
	require.Equal(t, "retweeter", page.Items[0].Retweet.By.Username)
	require.Equal(t, int64(7), page.Items[0].Retweet.ID)
	require.Equal(t, &pagination.Cursor{Time: retweetedAt, Kind: 1, ID: 7}, page.Next, "retweets page by the retweet, not the tweet")
}

func TestLikes_AreOnlyVisibleToTheirOwner(t *testing.T) {
	svc := feed.New(&dbtest.MockStore{})
	for _, viewer := range []*int64{nil, dbtest.Ptr(int64(2))} {
		_, err := svc.Likes(context.Background(), 1, pagination.Request{Limit: 10}, viewer)
		kind, ok := apperr.KindOf(err)
		require.True(t, ok)
		require.Equal(t, apperr.KindForbidden, kind)
	}
}

func TestPosts_UnknownUserIsNotFound(t *testing.T) {
	store := &dbtest.MockStore{
		GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
			return db.GetUserRow{}, pgxNoRows()
		},
	}
	_, err := feed.New(store).Posts(context.Background(), 99, pagination.Request{Limit: 10}, nil)
	kind, _ := apperr.KindOf(err)
	require.Equal(t, apperr.KindNotFound, kind)
}
