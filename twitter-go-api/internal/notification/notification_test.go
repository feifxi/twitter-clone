package notification_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	userID  int64
	evType  string
	payload any
}

func (p *recordingPublisher) Publish(_ context.Context, userID int64, eventType string, payload any) error {
	p.userID, p.evType, p.payload = userID, eventType, payload
	return nil
}

func previewStore(tweetID int64) *dbtest.MockStore {
	return &dbtest.MockStore{
		GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
			return []db.GetUsersByIDsRow{{User: db.User{ID: 2, Username: "actor"}}}, nil
		},
		GetTweetsByIDsFn: func(ctx context.Context, arg db.GetTweetsByIDsParams) ([]db.GetTweetsByIDsRow, error) {
			content := "tweet content"
			return []db.GetTweetsByIDsRow{{Tweet: db.Tweet{ID: tweetID, Content: &content}}}, nil
		},
	}
}

func TestService_List(t *testing.T) {
	tweetID := int64(10)
	store := previewStore(tweetID)
	store.ListNotificationsFn = func(ctx context.Context, arg db.ListNotificationsParams) ([]db.Notification, error) {
		return []db.Notification{{ID: 100, RecipientID: 1, ActorID: 2, Type: "LIKE", TweetID: &tweetID}}, nil
	}

	res, err := notification.New(store, &recordingPublisher{}).List(context.Background(), 1, pagination.Request{Limit: 10})

	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	require.Nil(t, res.Next)
	require.Equal(t, "actor", res.Items[0].Actor.Username)
	require.Equal(t, domain.NotificationLike, res.Items[0].Type)
	require.Equal(t, "tweet content", *res.Items[0].Tweet.Content)
	require.Nil(t, res.Items[0].OriginalTweet, "only replies carry the original post")
}

func TestService_ListReplyIncludesTheOriginalPostWithItsMedia(t *testing.T) {
	original := db.Tweet{ID: 10, Content: dbtest.Ptr("hello"), MediaType: dbtest.Ptr("IMAGE"), MediaURL: dbtest.Ptr("https://cdn/tweets/1/a.png")}
	reply := db.Tweet{ID: 11, Content: dbtest.Ptr("nice"), ParentID: &original.ID}
	store := &dbtest.MockStore{
		ListNotificationsFn: func(ctx context.Context, arg db.ListNotificationsParams) ([]db.Notification, error) {
			return []db.Notification{{ID: 1, RecipientID: 1, ActorID: 2, Type: "REPLY", TweetID: &reply.ID}}, nil
		},
		GetTweetsByIDsFn: func(ctx context.Context, arg db.GetTweetsByIDsParams) ([]db.GetTweetsByIDsRow, error) {
			byID := map[int64]db.Tweet{10: original, 11: reply}
			var rows []db.GetTweetsByIDsRow
			for _, id := range arg.TweetIds {
				rows = append(rows, db.GetTweetsByIDsRow{Tweet: byID[id]})
			}
			return rows, nil
		},
	}

	res, err := notification.New(store, &recordingPublisher{}).List(context.Background(), 1, pagination.Request{Limit: 10})

	require.NoError(t, err)
	item := res.Items[0]
	require.Equal(t, int64(11), item.Tweet.ID)
	require.Equal(t, int64(10), item.OriginalTweet.ID)
	require.Equal(t, "IMAGE", *item.OriginalTweet.MediaType)
	require.Equal(t, "https://cdn/tweets/1/a.png", *item.OriginalTweet.MediaURL)
}

func TestService_Counters(t *testing.T) {
	ctx := context.Background()

	t.Run("count_unread", func(t *testing.T) {
		store := &dbtest.MockStore{
			GetUnreadNotificationCountFn: func(ctx context.Context, id int64) (int64, error) { return 5, nil },
		}
		count, err := notification.New(store, nil).CountUnread(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, int64(5), count)
	})

	t.Run("mark_all_read", func(t *testing.T) {
		called := false
		store := &dbtest.MockStore{
			MarkAllNotificationsReadFn: func(ctx context.Context, id int64) error {
				called = true
				require.Equal(t, int64(1), id)
				return nil
			},
		}
		require.NoError(t, notification.New(store, nil).MarkAllRead(ctx, 1))
		require.True(t, called)
	})
}

func TestService_DispatchPublishesHydratedNotification(t *testing.T) {
	tweetID := int64(10)
	pub := &recordingPublisher{}

	notification.New(previewStore(tweetID), pub).Dispatch(db.Notification{ID: 7, RecipientID: 1, ActorID: 2, Type: "LIKE", TweetID: &tweetID})

	require.Equal(t, int64(1), pub.userID)
	require.Equal(t, notification.EventType, pub.evType)
	require.Equal(t, "actor", pub.payload.(domain.Notification).Actor.Username)
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("skips_self_action", func(t *testing.T) {
		n, err := notification.Create(ctx, &dbtest.MockStore{}, 1, 1, nil, domain.NotificationFollow)
		require.NoError(t, err)
		require.Zero(t, n.ID)
	})

	t.Run("deduped_insert_is_not_an_error", func(t *testing.T) {
		store := &dbtest.MockStore{
			CreateNotificationFn: func(ctx context.Context, arg db.CreateNotificationParams) (db.Notification, error) {
				return db.Notification{}, pgx.ErrNoRows
			},
		}
		n, err := notification.Create(ctx, store, 2, 1, nil, domain.NotificationFollow)
		require.NoError(t, err)
		require.Zero(t, n.ID)
	})
}
