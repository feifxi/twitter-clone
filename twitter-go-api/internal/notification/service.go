package notification

import (
	"context"
	"log/slog"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
)

// EventType is the realtime event name for a new notification.
const EventType = "notification"

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

func (s *Service) List(ctx context.Context, userID int64, req pagination.Request) (pagination.Page[domain.Notification], error) {
	rows, err := s.store.ListNotifications(ctx, db.ListNotificationsParams{
		RecipientID: userID,
		CursorTime:  req.AfterTime(),
		CursorID:    req.AfterID(),
		PageLimit:   req.FetchLimit(),
	})
	if err != nil {
		return pagination.Page[domain.Notification]{}, err
	}

	rows, more := pagination.Trim(rows, req.Limit)
	items, err := s.hydrate(ctx, rows)
	if err != nil {
		return pagination.Page[domain.Notification]{}, err
	}
	next := pagination.NextFrom(rows, more, func(n db.Notification) pagination.Cursor {
		return pagination.Cursor{Time: n.CreatedAt, ID: n.ID}
	})
	return pagination.Page[domain.Notification]{Items: items, Next: next}, nil
}

func (s *Service) CountUnread(ctx context.Context, userID int64) (int64, error) {
	return s.store.GetUnreadNotificationCount(ctx, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	return s.store.MarkAllNotificationsRead(ctx, userID)
}

// Dispatch pushes a freshly created notification to the recipient's open streams.
// Call it only after the creating transaction commits. Failures are logged, not
// returned: the notification is already stored and will show up on next fetch.
func (s *Service) Dispatch(n db.Notification) {
	if n.ID == 0 {
		return
	}

	// Detached from the request: the response may already be on its way.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, err := s.hydrate(ctx, []db.Notification{n})
	if err != nil || len(items) == 0 {
		slog.ErrorContext(ctx, "failed to hydrate notification; not pushed", "err", err, "notification_id", n.ID)
		return
	}
	if err := s.publisher.Publish(ctx, n.RecipientID, EventType, items[0]); err != nil {
		slog.ErrorContext(ctx, "failed to publish notification", "err", err, "notification_id", n.ID)
	}
}

type tweetPreview struct {
	content  *string
	mediaURL *string
	parentID *int64
}

// hydrate batch-loads actors and tweet previews (plus reply parents) for a page of notifications.
func (s *Service) hydrate(ctx context.Context, rows []db.Notification) ([]domain.Notification, error) {
	if len(rows) == 0 {
		return []domain.Notification{}, nil
	}

	actorIDs := make([]int64, 0, len(rows))
	tweetIDs := make([]int64, 0, len(rows))
	for _, n := range rows {
		actorIDs = append(actorIDs, n.ActorID)
		if n.TweetID != nil {
			tweetIDs = append(tweetIDs, *n.TweetID)
		}
	}

	actors, err := s.loadActors(ctx, actorIDs)
	if err != nil {
		return nil, err
	}
	previews, err := s.loadPreviews(ctx, tweetIDs)
	if err != nil {
		return nil, err
	}

	items := make([]domain.Notification, 0, len(rows))
	for _, n := range rows {
		item := domain.Notification{
			ID:          n.ID,
			RecipientID: n.RecipientID,
			Type:        domain.NotificationType(n.Type),
			IsRead:      n.IsRead,
			CreatedAt:   n.CreatedAt,
			Actor:       actors[n.ActorID],
			TweetID:     n.TweetID,
		}
		if n.TweetID != nil {
			if p, ok := previews[*n.TweetID]; ok {
				item.TweetContent = p.content
				item.TweetMediaURL = p.mediaURL
				if item.Type == domain.NotificationReply && p.parentID != nil {
					if parent, ok := previews[*p.parentID]; ok {
						item.OriginalTweetID = p.parentID
						item.OriginalTweetContent = parent.content
						item.OriginalTweetMediaURL = parent.mediaURL
					}
				}
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) loadActors(ctx context.Context, ids []int64) (map[int64]domain.User, error) {
	rows, err := s.store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{UserIds: uniq(ids)})
	if err != nil {
		return nil, err
	}
	actors := make(map[int64]domain.User, len(rows))
	for _, r := range rows {
		actors[r.User.ID] = domain.NewUser(r.User, r.IsFollowing)
	}
	return actors, nil
}

// loadPreviews loads the notified tweets and, for replies, the tweets they reply to.
func (s *Service) loadPreviews(ctx context.Context, tweetIDs []int64) (map[int64]tweetPreview, error) {
	previews := make(map[int64]tweetPreview)
	if len(tweetIDs) == 0 {
		return previews, nil
	}

	add := func(rows []db.GetTweetsByIDsRow) {
		for _, r := range rows {
			previews[r.Tweet.ID] = tweetPreview{content: r.Tweet.Content, mediaURL: r.Tweet.MediaURL, parentID: r.Tweet.ParentID}
		}
	}

	rows, err := s.store.GetTweetsByIDs(ctx, db.GetTweetsByIDsParams{TweetIds: uniq(tweetIDs)})
	if err != nil {
		return nil, err
	}
	add(rows)

	var parentIDs []int64
	for _, p := range previews {
		if p.parentID != nil {
			if _, loaded := previews[*p.parentID]; !loaded {
				parentIDs = append(parentIDs, *p.parentID)
			}
		}
	}
	if len(parentIDs) > 0 {
		parents, err := s.store.GetTweetsByIDs(ctx, db.GetTweetsByIDsParams{TweetIds: uniq(parentIDs)})
		if err != nil {
			return nil, err
		}
		add(parents)
	}
	return previews, nil
}

func uniq(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
