// Package tweet handles posting, deleting and interacting with tweets, and
// hydrates tweet rows into domain.Tweet for every feature that lists tweets.
package tweet

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/jackc/pgx/v5"
)

type objectStore interface {
	PublicURL(objectKey string) string
	KeyFromPublicURL(url string) (string, bool)
	DeleteObject(ctx context.Context, objectKey string) error
}

type embeddingPublisher interface {
	PublishEmbeddingEvent(ctx context.Context, tweetID int64, content string) error
}

type notifier interface {
	Dispatch(n db.Notification)
}

type Service struct {
	store      db.Store
	storage    objectStore
	embeddings embeddingPublisher
	notifier   notifier
}

func New(store db.Store, storage objectStore, embeddings embeddingPublisher, notifier notifier) *Service {
	return &Service{store: store, storage: storage, embeddings: embeddings, notifier: notifier}
}

type CreateInput struct {
	UserID   int64
	Content  *string
	ParentID *int64
	MediaKey *string // object key from a presigned upload; the media type is derived from it
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Tweet, error) {
	content := trimmedOrNil(in.Content)

	var mediaKey string
	var mediaType, mediaURL *string
	if in.MediaKey != nil && *in.MediaKey != "" {
		mediaKey = *in.MediaKey
		if err := media.ValidateOwnedKey(mediaKey, media.FolderTweets, in.UserID); err != nil {
			return domain.Tweet{}, err
		}
		typ, url := media.TypeForKey(mediaKey), s.storage.PublicURL(mediaKey)
		mediaType, mediaURL = &typ, &url
	}

	if content == nil && mediaURL == nil {
		return domain.Tweet{}, apperr.BadRequest("tweet must include text or media")
	}

	var created db.Tweet
	var pending db.Notification
	err := s.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		var err error
		created, err = q.CreateTweet(ctx, db.CreateTweetParams{
			UserID:    in.UserID,
			Content:   content,
			MediaType: mediaType,
			MediaURL:  mediaURL,
			ParentID:  in.ParentID,
		})
		if err != nil {
			return err
		}

		if in.ParentID != nil {
			if err := q.IncrementParentReplyCount(ctx, *in.ParentID); err != nil {
				return err
			}
			parent, err := q.GetTweet(ctx, db.GetTweetParams{ID: *in.ParentID})
			if err != nil {
				return db.NotFound(err, "parent tweet")
			}
			pending, err = notification.Create(ctx, q, parent.Tweet.UserID, in.UserID, &created.ID, domain.NotificationReply)
			if err != nil {
				return err
			}
		}

		if content != nil {
			for _, tag := range extractHashtags(*content) {
				h, err := q.UpsertHashtag(ctx, tag)
				if err != nil {
					return err
				}
				if err := q.LinkTweetHashtag(ctx, db.LinkTweetHashtagParams{TweetID: created.ID, HashtagID: h.ID}); err != nil {
					return err
				}
			}
		}
		return nil
	}, func() { s.notifier.Dispatch(pending) })
	if err != nil {
		if mediaKey != "" {
			_ = s.storage.DeleteObject(ctx, mediaKey)
		}
		return domain.Tweet{}, err
	}

	if content != nil {
		go s.publishEmbedding(created.ID, *content)
	}

	return s.Get(ctx, created.ID, &in.UserID)
}

// publishEmbedding runs detached from the request so it neither delays nor is cancelled with the response.
func (s *Service) publishEmbedding(tweetID int64, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.embeddings.PublishEmbeddingEvent(ctx, tweetID, content)
}

func (s *Service) Delete(ctx context.Context, userID, tweetID int64) error {
	t, err := s.getTweet(ctx, tweetID, nil)
	if err != nil {
		return err
	}
	if t.Tweet.UserID != userID {
		return apperr.Forbidden("you can only delete your own tweets")
	}

	mediaURLs, err := s.store.ListMediaURLsInThread(ctx, tweetID)
	if err != nil {
		return err
	}

	return s.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		// Read hashtag usage for the whole cascade (root + replies + retweets) before
		// deleting, because ON DELETE CASCADE removes the tweet_hashtags rows.
		hashtagUsage, err := q.ListHashtagUsageToDecrementForDeleteRoot(ctx, tweetID)
		if err != nil {
			return err
		}

		if t.Tweet.RetweetID != nil {
			_, err = q.DeleteRetweetByUser(ctx, db.DeleteRetweetByUserParams{UserID: userID, RetweetID: t.Tweet.RetweetID})
		} else {
			_, err = q.DeleteTweetByOwner(ctx, db.DeleteTweetByOwnerParams{ID: tweetID, UserID: userID})
		}
		if err != nil {
			return err
		}

		if t.Tweet.ParentID != nil {
			if err := q.DecrementParentReplyCount(ctx, *t.Tweet.ParentID); err != nil {
				return err
			}
		}

		for _, usage := range hashtagUsage {
			if err := q.DecrementHashtagUsageBy(ctx, db.DecrementHashtagUsageByParams{ID: usage.HashtagID, UsageCount: usage.DecrementBy}); err != nil {
				return err
			}
			if err := q.DeleteUnusedHashtag(ctx, usage.HashtagID); err != nil {
				return err
			}
		}
		return nil
	}, func() { s.deleteMedia(mediaURLs) })
}

// deleteMedia removes the thread's hosted media after the rows are gone; external URLs are skipped.
func (s *Service) deleteMedia(urls []*string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	seen := make(map[string]struct{}, len(urls))
	for _, url := range urls {
		if url == nil {
			continue
		}
		key, ok := s.storage.KeyFromPublicURL(*url)
		if !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		_ = s.storage.DeleteObject(ctx, key)
	}
}

func (s *Service) Get(ctx context.Context, tweetID int64, viewerID *int64) (domain.Tweet, error) {
	row, err := s.getTweet(ctx, tweetID, viewerID)
	if err != nil {
		return domain.Tweet{}, err
	}
	items, err := Hydrate(ctx, s.store, []db.GetTweetRow{row}, viewerID, func(r db.GetTweetRow) Row { return Row(r) })
	if err != nil {
		return domain.Tweet{}, err
	}
	return items[0], nil
}

func (s *Service) ListReplies(ctx context.Context, tweetID int64, page, size int32, viewerID *int64) ([]domain.Tweet, error) {
	if _, err := s.getTweet(ctx, tweetID, viewerID); err != nil {
		return nil, err
	}

	rows, err := s.store.ListTweetReplies(ctx, db.ListTweetRepliesParams{
		ParentID: &tweetID,
		Limit:    size,
		Offset:   page * size,
		ViewerID: viewerID,
	})
	if err != nil {
		return nil, err
	}
	return Hydrate(ctx, s.store, rows, viewerID, func(r db.ListTweetRepliesRow) Row { return Row(r) })
}

func (s *Service) Like(ctx context.Context, userID, tweetID int64) error {
	t, err := s.getTweet(ctx, tweetID, nil)
	if err != nil {
		return err
	}

	var pending db.Notification
	return s.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		liked, err := q.LikeTweet(ctx, db.LikeTweetParams{UserID: userID, TweetID: tweetID})
		if err != nil || !liked {
			return err
		}
		pending, err = notification.Create(ctx, q, t.Tweet.UserID, userID, &t.Tweet.ID, domain.NotificationLike)
		return err
	}, func() { s.notifier.Dispatch(pending) })
}

func (s *Service) Unlike(ctx context.Context, userID, tweetID int64) error {
	if _, err := s.getTweet(ctx, tweetID, nil); err != nil {
		return err
	}
	_, err := s.store.UnlikeTweet(ctx, db.UnlikeTweetParams{UserID: userID, TweetID: tweetID})
	return err
}

// Retweet is idempotent: retweeting again returns the existing retweet. Retweeting
// a retweet targets its original.
func (s *Service) Retweet(ctx context.Context, userID, tweetID int64) (domain.Tweet, error) {
	original, err := s.originalOf(ctx, tweetID)
	if err != nil {
		return domain.Tweet{}, err
	}

	var created db.CreateRetweetRow
	var pending db.Notification
	err = s.store.ExecTxAfterCommit(ctx, func(q db.Querier) error {
		var err error
		created, err = q.CreateRetweet(ctx, db.CreateRetweetParams{UserID: userID, RetweetID: &original.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING returned nothing: the retweet already exists.
			existing, err := q.GetUserRetweet(ctx, db.GetUserRetweetParams{UserID: userID, RetweetID: &original.ID})
			created = db.CreateRetweetRow(existing)
			return err
		}
		if err != nil {
			return err
		}
		pending, err = notification.Create(ctx, q, original.UserID, userID, &original.ID, domain.NotificationRetweet)
		return err
	}, func() { s.notifier.Dispatch(pending) })
	if err != nil {
		return domain.Tweet{}, err
	}

	return s.Get(ctx, created.ID, &userID)
}

func (s *Service) UndoRetweet(ctx context.Context, userID, tweetID int64) error {
	original, err := s.originalOf(ctx, tweetID)
	if err != nil {
		return err
	}

	_, err = s.store.DeleteRetweetByUser(ctx, db.DeleteRetweetByUserParams{UserID: userID, RetweetID: &original.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// originalOf resolves a retweet to the tweet it points at.
func (s *Service) originalOf(ctx context.Context, tweetID int64) (db.Tweet, error) {
	t, err := s.getTweet(ctx, tweetID, nil)
	if err != nil {
		return db.Tweet{}, err
	}
	if t.Tweet.RetweetID == nil {
		return t.Tweet, nil
	}
	original, err := s.getTweet(ctx, *t.Tweet.RetweetID, nil)
	return original.Tweet, err
}

func (s *Service) getTweet(ctx context.Context, tweetID int64, viewerID *int64) (db.GetTweetRow, error) {
	row, err := s.store.GetTweet(ctx, db.GetTweetParams{ID: tweetID, ViewerID: viewerID})
	return row, db.NotFound(err, "tweet")
}

func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
