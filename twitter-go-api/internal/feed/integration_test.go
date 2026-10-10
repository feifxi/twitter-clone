package feed_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/feed"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/stretchr/testify/require"
)

// collect pages through a list until the cursor runs out.
func collect(t *testing.T, limit int32, fetch func(pagination.Request) (pagination.Page[domain.Tweet], error)) []domain.Tweet {
	t.Helper()
	var all []domain.Tweet
	req := pagination.Request{Limit: limit}
	for range 100 {
		page, err := fetch(req)
		require.NoError(t, err)
		require.LessOrEqual(t, int32(len(page.Items)), limit)
		all = append(all, page.Items...)
		if page.Next == nil {
			return all
		}
		// Cursors travel through clients as strings.
		next, err := pagination.Decode(pagination.Encode(*page.Next))
		require.NoError(t, err)
		req.After = next
	}
	t.Fatal("pagination did not terminate")
	return nil
}

func TestFollowingTimeline_PagesTweetsAndRetweetsInOrder(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()

	viewer, author, retweeter := dbtest.NewUser(t, store), dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	for _, followee := range []int64{author.ID, retweeter.ID} {
		_, err := store.FollowUser(ctx, db.FollowUserParams{FollowerID: viewer.ID, FollowingID: followee})
		require.NoError(t, err)
	}

	var tweets []db.Tweet
	for _, text := range []string{"one", "two", "three", "four", "five"} {
		tweets = append(tweets, dbtest.NewTweet(t, store, author.ID, text))
	}
	// A reply never shows on the timeline; a retweet of an old tweet shows at the top.
	_, err := store.CreateTweet(ctx, db.CreateTweetParams{UserID: author.ID, Content: dbtest.Ptr("reply"), ParentID: &tweets[0].ID})
	require.NoError(t, err)
	_, err = store.CreateRetweet(ctx, db.CreateRetweetParams{UserID: retweeter.ID, TweetID: tweets[1].ID})
	require.NoError(t, err)

	svc := feed.New(store)
	items := collect(t, 2, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Following(ctx, viewer.ID, req)
	})

	require.Len(t, items, 6)
	require.Equal(t, tweets[1].ID, items[0].ID)
	require.NotNil(t, items[0].RetweetedBy)
	require.Equal(t, retweeter.ID, items[0].RetweetedBy.ID)
	for i, want := range []int64{tweets[4].ID, tweets[3].ID, tweets[2].ID, tweets[1].ID, tweets[0].ID} {
		require.Equal(t, want, items[i+1].ID, "position %d", i+1)
		require.Nil(t, items[i+1].RetweetedBy)
	}
}

func TestForYou_PagesWithoutDuplicatesAndRanksByEngagement(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()

	author, fan := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	quiet := dbtest.NewTweet(t, store, author.ID, "quiet")
	popular := dbtest.NewTweet(t, store, author.ID, "popular")
	_, err := store.LikeTweet(ctx, db.LikeTweetParams{UserID: fan.ID, TweetID: popular.ID})
	require.NoError(t, err)
	_, err = store.CreateRetweet(ctx, db.CreateRetweetParams{UserID: fan.ID, TweetID: popular.ID})
	require.NoError(t, err)

	svc := feed.New(store)
	items := collect(t, 3, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Global(ctx, req, &fan.ID)
	})

	seen := map[int64]int{}
	for i, it := range items {
		_, dup := seen[it.ID]
		require.False(t, dup, "tweet %d appears twice", it.ID)
		seen[it.ID] = i
	}
	require.Contains(t, seen, quiet.ID)
	require.Contains(t, seen, popular.ID)
	require.Less(t, seen[popular.ID], seen[quiet.ID], "engagement ranks the popular tweet higher")
	require.True(t, items[seen[popular.ID]].IsLiked)
	require.True(t, items[seen[popular.ID]].IsRetweeted)
}

func TestProfileTabs(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()

	owner, other := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	post := dbtest.NewTweet(t, store, owner.ID, "post")
	reply, err := store.CreateTweet(ctx, db.CreateTweetParams{UserID: owner.ID, Content: dbtest.Ptr("reply"), ParentID: &post.ID})
	require.NoError(t, err)
	media, err := store.CreateTweet(ctx, db.CreateTweetParams{UserID: owner.ID, MediaURL: dbtest.Ptr("https://cdn/x.png"), MediaType: dbtest.Ptr("IMAGE")})
	require.NoError(t, err)
	othersTweet := dbtest.NewTweet(t, store, other.ID, "liked")
	_, err = store.LikeTweet(ctx, db.LikeTweetParams{UserID: owner.ID, TweetID: othersTweet.ID})
	require.NoError(t, err)

	svc := feed.New(store)
	ids := func(items []domain.Tweet) []int64 {
		out := make([]int64, len(items))
		for i, it := range items {
			out[i] = it.ID
		}
		return out
	}

	posts := collect(t, 10, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Posts(ctx, owner.ID, req, nil)
	})
	require.Equal(t, []int64{media.ID, post.ID}, ids(posts), "posts exclude replies")

	replies := collect(t, 10, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Replies(ctx, owner.ID, req, nil)
	})
	require.Equal(t, []int64{reply.ID}, ids(replies))
	require.Equal(t, owner.Username, *replies[0].ParentUsername)

	mediaTab := collect(t, 10, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Media(ctx, owner.ID, req, nil)
	})
	require.Equal(t, []int64{media.ID}, ids(mediaTab))

	likes := collect(t, 10, func(req pagination.Request) (pagination.Page[domain.Tweet], error) {
		return svc.Likes(ctx, owner.ID, req, &owner.ID)
	})
	require.Equal(t, []int64{othersTweet.ID}, ids(likes))
	require.True(t, likes[0].IsLiked)
}
