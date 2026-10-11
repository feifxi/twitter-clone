package message_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type publishedEvent struct {
	userID  int64
	evType  string
	payload any
}

// recordingPublisher is safe for concurrent use (sends can run in parallel in tests).
type recordingPublisher struct {
	mu     sync.Mutex
	events []publishedEvent
}

func (p *recordingPublisher) Publish(_ context.Context, userID int64, eventType string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, publishedEvent{userID, eventType, payload})
	return nil
}

func TestService_SendMessageToUser(t *testing.T) {
	ctx := context.Background()
	senderID := int64(1)
	recipientID := int64(2)
	content := "Hello!"

	t.Run("success_existing_conversation", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: recipientID}}, nil
			},
			FindDirectConversationFn: func(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error) {
				return db.Conversation{ID: 10}, nil
			},
			CreateDirectMessageFn: func(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error) {
				return db.DirectMessage{ID: 100, ConversationID: 10, SenderID: senderID, Content: content, CreatedAt: time.Now()}, nil
			},
			TouchConversationFn: func(ctx context.Context, id int64) error {
				return nil
			},
			ListConversationParticipantIDsFn: func(ctx context.Context, id int64) ([]int64, error) {
				return []int64{senderID, recipientID}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: senderID, Username: "sender"}},
				}, nil
			},
		}

		pub := &recordingPublisher{}
		uc := message.New(store, pub)
		msg, err := uc.SendToUser(ctx, senderID, recipientID, content)

		require.NoError(t, err)
		require.Equal(t, int64(100), msg.ID)
		require.Equal(t, int64(10), msg.ConversationID)
		require.Equal(t, "sender", msg.Sender.Username)
		require.Len(t, pub.events, 2)
		for _, ev := range pub.events {
			require.Equal(t, message.EventType, ev.evType)
		}
		require.ElementsMatch(t, []int64{senderID, recipientID}, []int64{pub.events[0].userID, pub.events[1].userID})
	})

	t.Run("success_new_conversation", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: recipientID}}, nil
			},
			FindDirectConversationFn: func(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error) {
				return db.Conversation{}, pgx.ErrNoRows
			},
			CreateConversationFn: func(ctx context.Context) (db.Conversation, error) {
				return db.Conversation{ID: 20}, nil
			},
			AddConversationParticipantFn: func(ctx context.Context, arg db.AddConversationParticipantParams) error {
				return nil
			},
			CreateDirectMessageFn: func(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error) {
				return db.DirectMessage{ID: 200, ConversationID: 20, SenderID: senderID, Content: content, CreatedAt: time.Now()}, nil
			},
			TouchConversationFn: func(ctx context.Context, id int64) error {
				return nil
			},
			ListConversationParticipantIDsFn: func(ctx context.Context, id int64) ([]int64, error) {
				return []int64{senderID, recipientID}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: senderID, Username: "sender"}},
				}, nil
			},
		}

		uc := message.New(store, &recordingPublisher{})
		msg, err := uc.SendToUser(ctx, senderID, recipientID, content)

		require.NoError(t, err)
		require.Equal(t, int64(20), msg.ConversationID)
	})
}

func contents(msgs []domain.Message) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = m.Content
	}
	return out
}
