package notification

import (
	"context"
	"errors"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/jackc/pgx/v5"
)

// Create inserts a notification using q, so callers can make it part of their
// transaction. It returns a zero Notification (ID 0) when nothing was created:
// self-actions are skipped and repeats are deduped by idx_notifications_dedupe.
func Create(ctx context.Context, q db.Querier, recipientID, actorID int64, tweetID *int64, typ domain.NotificationType) (db.Notification, error) {
	if recipientID == actorID {
		return db.Notification{}, nil
	}

	n, err := q.CreateNotification(ctx, db.CreateNotificationParams{
		RecipientID: recipientID,
		ActorID:     actorID,
		TweetID:     tweetID,
		Type:        string(typ),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Notification{}, nil
	}
	return n, err
}
