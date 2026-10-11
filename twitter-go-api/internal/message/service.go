// Package message implements one-to-one direct messages.
package message

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/jackc/pgx/v5"
)

const maxContentLength = 2000

// EventType is the realtime event name for a new direct message.
const EventType = "dm"

type publisher interface {
	Publish(ctx context.Context, userID int64, eventType string, payload any) error
}

type Service struct {
	store     db.Store
	publisher publisher
}

func New(store db.Store, publisher publisher) *Service {
	return &Service{store: store, publisher: publisher}
}

func normalizeContent(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", apperr.BadRequest("message content is required")
	}
	// Characters, not bytes: matches the VARCHAR(2000) column (Thai is 3 bytes per character).
	if utf8.RuneCountInString(trimmed) > maxContentLength {
		return "", apperr.BadRequest("message content exceeds 2000 characters")
	}
	return trimmed, nil
}

func (s *Service) fetchSender(ctx context.Context, senderID int64) (domain.User, error) {
	rows, err := s.store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{
		ViewerID: &senderID,
		UserIds:  []int64{senderID},
	})
	if err != nil {
		return domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.User{}, apperr.Internal("sender not found", nil)
	}
	return domain.NewUser(rows[0].User, rows[0].IsFollowing), nil
}

// Conversations lists the user's conversations, most recently active first.
func (s *Service) Conversations(ctx context.Context, userID int64, req pagination.Request) (pagination.Page[domain.Conversation], error) {
	rows, err := s.store.ListUserConversations(ctx, db.ListUserConversationsParams{
		UserID:     userID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return pagination.Page[domain.Conversation]{}, err
	}
	rows, more := pagination.Trim(rows, req.Limit)

	ids := make([]int64, 0, len(rows)*2)
	for _, r := range rows {
		ids = append(ids, r.User.ID, r.LastSenderID)
	}
	users, err := s.loadUsers(ctx, ids, userID)
	if err != nil {
		return pagination.Page[domain.Conversation]{}, err
	}

	items := make([]domain.Conversation, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.Conversation{
			ID:        r.ConversationID,
			Peer:      users[r.User.ID],
			UpdatedAt: r.ConversationUpdatedAt,
			LastMessage: domain.Message{
				ID:             r.LastMessageID,
				ConversationID: r.ConversationID,
				Sender:         users[r.LastSenderID],
				Content:        r.LastMessageContent,
				CreatedAt:      r.LastMessageCreatedAt,
			},
		})
	}
	next := pagination.NextFrom(rows, more, func(r db.ListUserConversationsRow) pagination.Cursor {
		return pagination.Cursor{Time: r.LastMessageCreatedAt, ID: r.ConversationID}
	})
	return pagination.Page[domain.Conversation]{Items: items, Next: next}, nil
}

// Messages pages backwards through a conversation: each page holds older messages,
// returned oldest-first so they render top to bottom.
func (s *Service) Messages(ctx context.Context, userID, conversationID int64, req pagination.Request) (pagination.Page[domain.Message], error) {
	allowed, err := s.store.IsConversationParticipant(ctx, db.IsConversationParticipantParams{ConversationID: conversationID, UserID: userID})
	if err != nil {
		return pagination.Page[domain.Message]{}, err
	}
	if !allowed {
		return pagination.Page[domain.Message]{}, apperr.Forbidden("not allowed to access this conversation")
	}

	rows, err := s.store.ListConversationMessages(ctx, db.ListConversationMessagesParams{
		ConversationID: conversationID,
		CursorTime:     req.AfterTime(),
		CursorID:       req.AfterID(),
		PageLimit:      req.FetchLimit(),
	})
	if err != nil {
		return pagination.Page[domain.Message]{}, err
	}
	rows, more := pagination.Trim(rows, req.Limit)
	// rows are newest-first, so the last one is the oldest: the next page continues from it.
	next := pagination.NextFrom(rows, more, func(m db.DirectMessage) pagination.Cursor {
		return pagination.Cursor{Time: m.CreatedAt, ID: m.ID}
	})

	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.SenderID)
	}
	users, err := s.loadUsers(ctx, ids, userID)
	if err != nil {
		return pagination.Page[domain.Message]{}, err
	}

	items := make([]domain.Message, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		items = append(items, domain.Message{
			ID:             r.ID,
			ConversationID: r.ConversationID,
			Sender:         users[r.SenderID],
			Content:        r.Content,
			CreatedAt:      r.CreatedAt,
		})
	}
	return pagination.Page[domain.Message]{Items: items, Next: next}, nil
}

func (s *Service) loadUsers(ctx context.Context, ids []int64, viewerID int64) (map[int64]domain.User, error) {
	users := make(map[int64]domain.User, len(ids))
	if len(ids) == 0 {
		return users, nil
	}
	rows, err := s.store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{UserIds: ids, ViewerID: &viewerID})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		users[r.User.ID] = domain.NewUser(r.User, r.IsFollowing)
	}
	return users, nil
}

// SendToUser sends to the direct conversation with recipientID, creating it on first contact.
func (s *Service) SendToUser(ctx context.Context, senderID, recipientID int64, content string) (domain.Message, error) {
	if senderID == recipientID {
		return domain.Message{}, apperr.BadRequest("cannot message yourself")
	}
	normalized, err := normalizeContent(content)
	if err != nil {
		return domain.Message{}, err
	}
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: recipientID}); err != nil {
		return domain.Message{}, db.NotFound(err, "user")
	}

	var created db.DirectMessage
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		conversationID, err := findOrCreateDirectConversation(ctx, q, senderID, recipientID)
		if err != nil {
			return err
		}
		created, err = insertMessage(ctx, q, conversationID, senderID, normalized)
		return err
	})
	if err != nil {
		return domain.Message{}, err
	}
	return s.delivered(ctx, created)
}

// SendToConversation sends to an existing conversation the sender belongs to.
func (s *Service) SendToConversation(ctx context.Context, senderID, conversationID int64, content string) (domain.Message, error) {
	normalized, err := normalizeContent(content)
	if err != nil {
		return domain.Message{}, err
	}

	allowed, err := s.store.IsConversationParticipant(ctx, db.IsConversationParticipantParams{ConversationID: conversationID, UserID: senderID})
	if err != nil {
		return domain.Message{}, err
	}
	if !allowed {
		return domain.Message{}, apperr.Forbidden("not allowed to send messages to this conversation")
	}

	var created db.DirectMessage
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		created, err = insertMessage(ctx, q, conversationID, senderID, normalized)
		return err
	})
	if err != nil {
		return domain.Message{}, err
	}
	return s.delivered(ctx, created)
}

func findOrCreateDirectConversation(ctx context.Context, q db.Querier, userA, userB int64) (int64, error) {
	lo, hi := min(userA, userB), max(userA, userB)
	if err := q.LockDirectConversationPair(ctx, fmt.Sprintf("dm:%d:%d", lo, hi)); err != nil {
		return 0, err
	}

	conversation, err := q.FindDirectConversation(ctx, db.FindDirectConversationParams{UserID: userA, UserID_2: userB})
	if err == nil {
		return conversation.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	conversation, err = q.CreateConversation(ctx)
	if err != nil {
		return 0, err
	}
	for _, userID := range []int64{userA, userB} {
		if err := q.AddConversationParticipant(ctx, db.AddConversationParticipantParams{ConversationID: conversation.ID, UserID: userID}); err != nil {
			return 0, err
		}
	}
	return conversation.ID, nil
}

// insertMessage also bumps the conversation so it sorts first in the inbox.
func insertMessage(ctx context.Context, q db.Querier, conversationID, senderID int64, content string) (db.DirectMessage, error) {
	msg, err := q.CreateDirectMessage(ctx, db.CreateDirectMessageParams{ConversationID: conversationID, SenderID: senderID, Content: content})
	if err != nil {
		return db.DirectMessage{}, err
	}
	return msg, q.TouchConversation(ctx, conversationID)
}

// delivered builds the response message and pushes it to every participant's open streams
// (the sender's other devices included). Push failures are logged: the message is stored.
func (s *Service) delivered(ctx context.Context, msg db.DirectMessage) (domain.Message, error) {
	sender, err := s.fetchSender(ctx, msg.SenderID)
	if err != nil {
		return domain.Message{}, err
	}
	out := domain.Message{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Sender:         sender,
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt,
	}

	participants, err := s.store.ListConversationParticipantIDs(ctx, msg.ConversationID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list participants; message not pushed", "err", err, "message_id", msg.ID)
		return out, nil
	}
	for _, userID := range participants {
		if err := s.publisher.Publish(ctx, userID, EventType, out); err != nil {
			slog.ErrorContext(ctx, "failed to publish message", "err", err, "message_id", msg.ID, "user_id", userID)
		}
	}
	return out, nil
}
