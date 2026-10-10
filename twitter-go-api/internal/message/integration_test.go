package message_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/stretchr/testify/require"
)

func TestMessages_PageBackwardsInReadingOrder(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	alice, bob := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
	pub := &recordingPublisher{}
	svc := message.New(store, pub)

	var conversationID int64
	for i := range 5 {
		msg, err := svc.SendToUser(ctx, alice.ID, bob.ID, fmt.Sprintf("m%d", i))
		require.NoError(t, err)
		conversationID = msg.ConversationID
	}
	require.Len(t, pub.events, 10, "every message goes to both participants")

	latest, err := svc.Messages(ctx, bob.ID, conversationID, pagination.Request{Limit: 3})
	require.NoError(t, err)
	require.Equal(t, []string{"m2", "m3", "m4"}, contents(latest.Items), "newest page, oldest-first")
	require.NotNil(t, latest.Next)

	older, err := svc.Messages(ctx, bob.ID, conversationID, pagination.Request{After: latest.Next, Limit: 3})
	require.NoError(t, err)
	require.Equal(t, []string{"m0", "m1"}, contents(older.Items))
	require.Nil(t, older.Next)

	inbox, err := svc.Conversations(ctx, bob.ID, pagination.Request{Limit: 10})
	require.NoError(t, err)
	require.Len(t, inbox.Items, 1)
	require.Equal(t, alice.ID, inbox.Items[0].Peer.ID)
	require.Equal(t, "m4", inbox.Items[0].LastMessage.Content)
}
