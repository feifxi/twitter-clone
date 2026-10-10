package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/jackc/pgx/v5"
)

var hashtagRegex = regexp.MustCompile(`(?i)(?:^|\s)#([a-z0-9_]+)`)

func extractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{})
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		tag := strings.ToLower(strings.TrimSpace(m[1]))
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}

	return result
}

func buildTSQuery(raw string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, raw)
	parts := strings.Fields(clean)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " & ")
}

// createNotification inserts a notification row using the provided Queries handle.
// Use inside ExecTx to ensure the row is part of the transaction.
// Returns the notification (for dispatch after commit) or zero value if skipped.
func createNotification(ctx context.Context, q db.Querier, recipientID, actorID int64, tweetID *int64, typ string) (db.Notification, error) {
	if recipientID == actorID {
		return db.Notification{}, nil
	}

	arg := db.CreateNotificationParams{
		RecipientID: recipientID,
		ActorID:     actorID,
		Type:        typ,
		TweetID:     nil,
	}
	if tweetID != nil {
		arg.TweetID = tweetID
	}

	notification, err := q.CreateNotification(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) {
		// Deduped by idx_notifications_dedupe (e.g. like → unlike → like).
		return db.Notification{}, nil
	}
	return notification, err
}

// dispatchNotification pushes a notification via SSE.
// Call ONLY after the transaction has committed successfully.
func dispatchNotification(publishNotification func(db.Notification), notification db.Notification) {
	if notification.ID == 0 {
		return // was skipped (self-notification)
	}
	if publishNotification != nil {
		publishNotification(notification)
	}
}
