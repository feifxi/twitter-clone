package notification_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/stretchr/testify/require"
)

func TestCreate_DedupesRepeatedActions(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	recipient, actor := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	tw := dbtest.NewTweet(t, store, recipient.ID, "like me")

	first, err := notification.Create(ctx, store, recipient.ID, actor.ID, &tw.ID, domain.NotificationLike)
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	again, err := notification.Create(ctx, store, recipient.ID, actor.ID, &tw.ID, domain.NotificationLike)
	require.NoError(t, err)
	require.Zero(t, again.ID, "like -> unlike -> like must not notify twice")

	// Follows have no tweet; NULLS NOT DISTINCT still dedupes them.
	for range 2 {
		_, err := notification.Create(ctx, store, recipient.ID, actor.ID, nil, domain.NotificationFollow)
		require.NoError(t, err)
	}

	page, err := notification.New(store, nil).List(ctx, recipient.ID, pagination.Request{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, domain.NotificationFollow, page.Items[0].Type, "newest first")
	require.Equal(t, actor.Username, page.Items[0].Actor.Username)
	unread, err := store.GetUnreadNotificationCount(ctx, recipient.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), unread)

}
