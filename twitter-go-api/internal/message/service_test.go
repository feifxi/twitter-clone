package message_test

import (
	"context"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type mockMessageStore struct {
	db.Querier
	listUserConversationsFn          func(ctx context.Context, arg db.ListUserConversationsParams) ([]db.ListUserConversationsRow, error)
	getUsersByIDsFn                  func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error)
	isConversationParticipantFn      func(ctx context.Context, arg db.IsConversationParticipantParams) (bool, error)
	listConversationMessagesFn       func(ctx context.Context, arg db.ListConversationMessagesParams) ([]db.DirectMessage, error)
	getUserFn                        func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error)
	findDirectConversationFn         func(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error)
	createConversationFn             func(ctx context.Context) (db.Conversation, error)
	addConversationParticipantFn     func(ctx context.Context, arg db.AddConversationParticipantParams) error
	createDirectMessageFn            func(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error)
	touchConversationFn              func(ctx context.Context, conversationID int64) error
	listConversationParticipantIDsFn func(ctx context.Context, conversationID int64) ([]int64, error)
}

func (m *mockMessageStore) ListUserConversations(ctx context.Context, arg db.ListUserConversationsParams) ([]db.ListUserConversationsRow, error) {
	return m.listUserConversationsFn(ctx, arg)
}
func (m *mockMessageStore) GetUsersByIDs(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
	return m.getUsersByIDsFn(ctx, arg)
}
func (m *mockMessageStore) IsConversationParticipant(ctx context.Context, arg db.IsConversationParticipantParams) (bool, error) {
	return m.isConversationParticipantFn(ctx, arg)
}
func (m *mockMessageStore) ListConversationMessages(ctx context.Context, arg db.ListConversationMessagesParams) ([]db.DirectMessage, error) {
	return m.listConversationMessagesFn(ctx, arg)
}
func (m *mockMessageStore) GetUser(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
	return m.getUserFn(ctx, arg)
}
func (m *mockMessageStore) FindDirectConversation(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error) {
	return m.findDirectConversationFn(ctx, arg)
}
func (m *mockMessageStore) CreateConversation(ctx context.Context) (db.Conversation, error) {
	return m.createConversationFn(ctx)
}
func (m *mockMessageStore) AddConversationParticipant(ctx context.Context, arg db.AddConversationParticipantParams) error {
	return m.addConversationParticipantFn(ctx, arg)
}
func (m *mockMessageStore) CreateDirectMessage(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error) {
	return m.createDirectMessageFn(ctx, arg)
}
func (m *mockMessageStore) TouchConversation(ctx context.Context, conversationID int64) error {
	return m.touchConversationFn(ctx, conversationID)
}
func (m *mockMessageStore) ListConversationParticipantIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	return m.listConversationParticipantIDsFn(ctx, conversationID)
}

func (m *mockMessageStore) ExecTx(ctx context.Context, fn func(db.Querier) error) error {
	return fn(m)
}
func (m *mockMessageStore) ExecTxAfterCommit(ctx context.Context, fn func(db.Querier) error, afterCommit func()) error {
	if err := fn(m); err != nil {
		return err
	}
	if afterCommit != nil {
		afterCommit()
	}
	return nil
}
func (m *mockMessageStore) Ping(ctx context.Context) error {
	return nil
}

func TestService_SendMessageToUser(t *testing.T) {
	ctx := context.Background()
	senderID := int64(1)
	recipientID := int64(2)
	content := "Hello!"

	t.Run("success_existing_conversation", func(t *testing.T) {
		store := &mockMessageStore{
			getUserFn: func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
				return db.GetUserRow{User: db.User{ID: recipientID}}, nil
			},
			findDirectConversationFn: func(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error) {
				return db.Conversation{ID: 10}, nil
			},
			createDirectMessageFn: func(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error) {
				return db.DirectMessage{ID: 100, ConversationID: 10, SenderID: senderID, Content: content, CreatedAt: time.Now()}, nil
			},
			touchConversationFn: func(ctx context.Context, id int64) error {
				return nil
			},
			listConversationParticipantIDsFn: func(ctx context.Context, id int64) ([]int64, error) {
				return []int64{senderID, recipientID}, nil
			},
			getUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{
					{User: db.User{ID: senderID, Username: "sender"}},
				}, nil
			},
		}

		uc := message.New(store)
		msg, participants, err := uc.SendToUser(ctx, senderID, recipientID, content)

		require.NoError(t, err)
		require.Equal(t, int64(100), msg.ID)
		require.Equal(t, int64(10), msg.ConversationID)
		require.Equal(t, "sender", msg.Sender.Username)
		require.ElementsMatch(t, []int64{senderID, recipientID}, participants)
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

		uc := message.New(store)
		msg, _, err := uc.SendToUser(ctx, senderID, recipientID, content)

		require.NoError(t, err)
		require.Equal(t, int64(20), msg.ConversationID)
	})
}
