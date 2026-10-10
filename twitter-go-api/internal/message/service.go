// Package message implements one-to-one direct messages.
package message

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
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
	if len(trimmed) > maxContentLength {
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

func (s *Service) Conversations(ctx context.Context, userID int64, page, size int32) ([]domain.Conversation, error) {
	rows, err := s.store.ListUserConversations(ctx, db.ListUserConversationsParams{
		UserID: userID,
		Limit:  size,
		Offset: page * size,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.Conversation{}, nil
	}

	idSet := make(map[int64]struct{}, len(rows)*2)
	for _, r := range rows {
		idSet[r.User.ID] = struct{}{}
		idSet[r.LastSenderID] = struct{}{}
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	viewerID := userID
	users, err := s.store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{
		ViewerID: &viewerID,
		UserIds:  ids,
	})
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]domain.User, len(users))
	for _, row := range users {
		userMap[row.User.ID] = domain.NewUser(row.User, row.IsFollowing)
	}

	items := make([]domain.Conversation, 0, len(rows))
	for _, r := range rows {
		peer, ok := userMap[r.User.ID]
		if !ok {
			continue
		}
		lastSender, ok := userMap[r.LastSenderID]
		if !ok {
			continue
		}

		items = append(items, domain.Conversation{
			ID:        r.ConversationID,
			Peer:      peer,
			UpdatedAt: r.ConversationUpdatedAt,
			LastMessage: domain.Message{
				ID:             r.LastMessageID,
				ConversationID: r.ConversationID,
				Sender:         lastSender,
				Content:        r.LastMessageContent,
				CreatedAt:      r.LastMessageCreatedAt,
			},
		})
	}

	return items, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID int64, page, size int32) ([]domain.Message, error) {
	allowed, err := s.store.IsConversationParticipant(ctx, db.IsConversationParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperr.Forbidden("not allowed to access this conversation")
	}

	rows, err := s.store.ListConversationMessages(ctx, db.ListConversationMessagesParams{
		ConversationID: conversationID,
		Limit:          size,
		Offset:         page * size,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.Message{}, nil
	}

	ids := make([]int64, 0, len(rows))
	seen := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.SenderID]; ok {
			continue
		}
		seen[row.SenderID] = struct{}{}
		ids = append(ids, row.SenderID)
	}

	viewerID := userID
	users, err := s.store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{
		ViewerID: &viewerID,
		UserIds:  ids,
	})
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]domain.User, len(users))
	for _, row := range users {
		userMap[row.User.ID] = domain.NewUser(row.User, row.IsFollowing)
	}

	items := make([]domain.Message, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i] // rows are DESC in SQL; reverse to ASC for chat rendering.
		sender, ok := userMap[row.SenderID]
		if !ok {
			continue
		}

		items = append(items, domain.Message{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			Sender:         sender,
			Content:        row.Content,
			CreatedAt:      row.CreatedAt,
		})
	}

	return items, nil
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
