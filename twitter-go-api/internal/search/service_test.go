package search_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/search"
	"github.com/stretchr/testify/require"
)

func TestService_SearchUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := &dbtest.MockStore{
			SearchUsersFn: func(ctx context.Context, arg db.SearchUsersParams) ([]db.SearchUsersRow, error) {
				return []db.SearchUsersRow{
					{User: db.User{ID: 1, Username: "founduser"}},
				}, nil
			},
		}

		uc := search.New(store)
		res, err := uc.Users(ctx, "query", 0, 10, nil)

		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "founduser", res[0].Username)
	})

	t.Run("empty_query", func(t *testing.T) {
		uc := search.New(nil)
		res, err := uc.Users(ctx, "  ", 0, 10, nil)
		require.NoError(t, err)
		require.Empty(t, res)
	})
}

func TestService_SearchTweets(t *testing.T) {
	ctx := context.Background()

	t.Run("hashtag_search", func(t *testing.T) {
		store := &dbtest.MockStore{
			SearchTweetsByHashtagFn: func(ctx context.Context, arg db.SearchTweetsByHashtagParams) ([]db.SearchTweetsByHashtagRow, error) {
				return []db.SearchTweetsByHashtagRow{
					{Tweet: db.Tweet{ID: 10, UserID: 100}},
				}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: 100, Username: "author"}},
				}, nil
			},
		}

		uc := search.New(store)
		res, err := uc.Tweets(ctx, "#Golang", 0, 10, nil)

		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, int64(10), res[0].ID)
	})

	t.Run("fulltext_search", func(t *testing.T) {
		store := &dbtest.MockStore{
			SearchTweetsFullTextFn: func(ctx context.Context, arg db.SearchTweetsFullTextParams) ([]db.SearchTweetsFullTextRow, error) {
				return []db.SearchTweetsFullTextRow{
					{Tweet: db.Tweet{ID: 20, UserID: 200}},
				}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: 200, Username: "author2"}},
				}, nil
			},
		}

		uc := search.New(store)
		res, err := uc.Tweets(ctx, "hello world", 0, 10, nil)

		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, int64(20), res[0].ID)
	})
}

func TestService_SearchHashtags(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := &dbtest.MockStore{
			SearchHashtagsByPrefixFn: func(ctx context.Context, arg db.SearchHashtagsByPrefixParams) ([]db.Hashtag, error) {
				return []db.Hashtag{{Text: "golang"}}, nil
			},
		}

		uc := search.New(store)
		res, err := uc.Hashtags(ctx, "go", 5)

		require.NoError(t, err)
		require.Len(t, res, 1)
		require.Equal(t, "golang", res[0].Text)
	})
}
