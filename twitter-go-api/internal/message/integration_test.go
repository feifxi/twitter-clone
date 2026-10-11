package message_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"

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

// Both users sending their first message at the same moment must share one conversation.
func TestSendToUser_ConcurrentFirstMessagesShareOneConversation(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	svc := message.New(store, &recordingPublisher{})

	for range 10 {
		a, b := dbtest.NewUser(t, store), dbtest.NewUser(t, store)
		var wg sync.WaitGroup
		ids := make([]int64, 2)
		errs := make([]error, 2)
		for i, pair := range [][2]int64{{a.ID, b.ID}, {b.ID, a.ID}} {
			wg.Go(func() {
				msg, err := svc.SendToUser(ctx, pair[0], pair[1], "hi")
				ids[i], errs[i] = msg.ConversationID, err
			})
		}
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		require.Equal(t, ids[0], ids[1], "both messages land in the same conversation")
	}
}

func TestSendToUser_LimitCountsCharactersNotBytes(t *testing.T) {
	store := dbtest.OpenTestDB(t)
	ctx := context.Background()
	svc := message.New(store, &recordingPublisher{})
	a, b := dbtest.NewUser(t, store), dbtest.NewUser(t, store)

	thai := strings.Repeat("ก", 2000) // 2000 characters, 6000 bytes
	_, err := svc.SendToUser(ctx, a.ID, b.ID, thai)
	require.NoError(t, err)

	_, err = svc.SendToUser(ctx, a.ID, b.ID, thai+"ก")
	kind, _ := apperr.KindOf(err)
	require.Equal(t, apperr.KindBadRequest, kind)
}
