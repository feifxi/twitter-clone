package tweet_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/storage/storagetest"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/stretchr/testify/require"
)

func TestRetweet_IsIdempotentAndKeepsCountsRight(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	author, fan := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	tw := dbtest.NewTweet(t, store, author.ID, "retweet me")
	svc := tweet.New(store, &storagetest.Fake{}, &mockEmbeddingPublisher{}, noopNotifier{})

	for range 2 {
		got, err := svc.Retweet(ctx, fan.ID, tw.ID)
		require.NoError(t, err)
		require.Equal(t, int32(1), got.RetweetCount)
		require.True(t, got.IsRetweeted)
	}

	for range 2 {
		require.NoError(t, svc.UndoRetweet(ctx, fan.ID, tw.ID))
	}
	got, err := svc.Get(ctx, tw.ID, &fan.ID)
	require.NoError(t, err)
	require.Zero(t, got.RetweetCount)
	require.False(t, got.IsRetweeted)
}

func TestDelete_RemovesThreadAndUnusedHashtags(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	author := dbtest.NewUser(t, store)
	svc := tweet.New(store, &storagetest.Fake{}, &mockEmbeddingPublisher{}, noopNotifier{})
	tag := "deltest" + author.Username[2:8]

	root, err := svc.Create(ctx, tweet.CreateInput{UserID: author.ID, Content: dbtest.Ptr("root #" + tag)})
	require.NoError(t, err)
	reply, err := svc.Create(ctx, tweet.CreateInput{UserID: author.ID, Content: dbtest.Ptr("reply #" + tag), ParentID: &root.ID})
	require.NoError(t, err)
	require.Equal(t, root.ID, *reply.ParentID)

	parent, err := svc.Get(ctx, root.ID, nil)
	require.NoError(t, err)
	require.Equal(t, int32(1), parent.ReplyCount)

	require.NoError(t, svc.Delete(ctx, author.ID, root.ID))

	_, err = svc.Get(ctx, reply.ID, nil)
	require.Error(t, err, "replies are deleted with their thread")
	tags, err := store.SearchHashtagsByPrefix(ctx, db.SearchHashtagsByPrefixParams{Prefix: tag, ResultLimit: 5})
	require.NoError(t, err)
	require.Empty(t, tags, "a hashtag used only by the deleted thread is removed")
}
