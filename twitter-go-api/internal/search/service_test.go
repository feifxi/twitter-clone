package search_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/search"
	"github.com/stretchr/testify/require"
)

func TestUsers_PagesByOffset(t *testing.T) {
	var got db.SearchUsersParams
	store := &dbtest.MockStore{
		SearchUsersFn: func(ctx context.Context, arg db.SearchUsersParams) ([]db.SearchUsersRow, error) {
			got = arg
			return []db.SearchUsersRow{{User: db.User{ID: 1, Username: "a"}}, {User: db.User{ID: 2, Username: "b"}}}, nil
		},
	}

	page, err := search.New(store).Users(context.Background(), " al ", pagination.Request{After: &pagination.Cursor{Offset: 10}, Limit: 1}, nil)

	require.NoError(t, err)
	require.Equal(t, "al", *got.Query)
	require.Equal(t, int32(10), got.PageOffset)
	require.Len(t, page.Items, 1)
	require.Equal(t, int32(11), page.Next.Offset)
}

func TestUsers_BlankQueryReturnsEmptyPage(t *testing.T) {
	page, err := search.New(nil).Users(context.Background(), "  ", pagination.Request{Limit: 10}, nil)
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Nil(t, page.Next)
}

func TestTweets_RoutesHashtagAndFullText(t *testing.T) {
	ctx := context.Background()
	var tag, tsQuery string
	store := &dbtest.MockStore{
		SearchTweetsByHashtagFn: func(ctx context.Context, arg db.SearchTweetsByHashtagParams) ([]db.SearchTweetsByHashtagRow, error) {
			tag = arg.Tag
			return []db.SearchTweetsByHashtagRow{{Tweet: db.Tweet{ID: 10, UserID: 1}}}, nil
		},
		SearchTweetsFullTextFn: func(ctx context.Context, arg db.SearchTweetsFullTextParams) ([]db.SearchTweetsFullTextRow, error) {
			tsQuery = arg.Query
			return []db.SearchTweetsFullTextRow{{Tweet: db.Tweet{ID: 20, UserID: 1}}}, nil
		},
		GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
			return []db.GetUsersByIDsRow{{User: db.User{ID: 1}}}, nil
		},
	}
	svc := search.New(store)

	byTag, err := svc.Tweets(ctx, "#Golang", pagination.Request{Limit: 10}, nil)
	require.NoError(t, err)
	require.Equal(t, "Golang", tag)
	require.Equal(t, int64(10), byTag.Items[0].ID)

	byText, err := svc.Tweets(ctx, "hello, world!", pagination.Request{Limit: 10}, nil)
	require.NoError(t, err)
	require.Equal(t, "hello & world", tsQuery)
	require.Equal(t, int64(20), byText.Items[0].ID)
}

func TestHashtags_StripsLeadingHash(t *testing.T) {
	var prefix string
	store := &dbtest.MockStore{
		SearchHashtagsByPrefixFn: func(ctx context.Context, arg db.SearchHashtagsByPrefixParams) ([]db.Hashtag, error) {
			prefix = arg.Prefix
			return []db.Hashtag{{ID: 1, Text: "golang"}}, nil
		},
	}
	res, err := search.New(store).Hashtags(context.Background(), "#go", 5)
	require.NoError(t, err)
	require.Equal(t, "go", prefix)
	require.Equal(t, "golang", res[0].Text)
}
