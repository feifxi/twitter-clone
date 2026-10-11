package tweet_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/db/dbtest"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/storage/storagetest"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/stretchr/testify/require"
)

type noopNotifier struct{}

func (noopNotifier) Dispatch(db.Notification) {}

func TestService_CreateTweet_Validation(t *testing.T) {
	uc := tweet.New(tweet.Config{}, &dbtest.MockStore{}, &storagetest.Fake{}, noopNotifier{})

	t.Run("empty_tweet", func(t *testing.T) {
		_, err := uc.Create(context.Background(), tweet.CreateInput{
			UserID: 1,
		})
		if err == nil {
			t.Fatal("expected error for empty tweet")
		}
		if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
			t.Fatalf("expected bad request, got %v", err)
		}
	})

	for name, key := range map[string]string{
		"no_folder":         "invalid-no-slash",
		"another_users_key": "tweets/2/0b1c.png",
		"wrong_folder":      "avatars/1/0b1c.png",
		"external_url":      "https://evil.example/tweets/1/x.png",
		"path_traversal":    "tweets/1/../2/x.png",
		"disallowed_ext":    "tweets/1/x.exe",
	} {
		t.Run("rejects_"+name, func(t *testing.T) {
			_, err := uc.Create(context.Background(), tweet.CreateInput{
				UserID:   1,
				MediaKey: dbtest.Ptr(key),
			})
			if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
				t.Fatalf("expected bad request for %q, got %v", key, err)
			}
		})
	}

	t.Run("derives_media_type_from_owned_key", func(t *testing.T) {
		var created db.CreateTweetParams
		store := &dbtest.MockStore{
			CreateTweetFn: func(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error) {
				created = arg
				return db.Tweet{ID: 10, UserID: arg.UserID, MediaURL: arg.MediaURL}, nil
			},
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 1}}}, nil
			},
		}
		uc := tweet.New(tweet.Config{}, store, &storagetest.Fake{}, noopNotifier{})

		_, err := uc.Create(context.Background(), tweet.CreateInput{UserID: 1, MediaKey: dbtest.Ptr("tweets/1/clip.mp4")})
		require.NoError(t, err)
		require.Equal(t, domain.MediaVideo, *created.MediaType)
		require.Equal(t, storagetest.BaseURL+"/tweets/1/clip.mp4", *created.MediaURL)
	})

	t.Run("success_with_hashtags", func(t *testing.T) {
		hashtagCalled := false
		linkCalled := false

		successStore := &dbtest.MockStore{
			CreateTweetFn: func(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error) {
				return db.Tweet{ID: 10, UserID: arg.UserID, Content: arg.Content}, nil
			},
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1, Content: dbtest.Ptr("Hello #world")}}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 1, Username: "testuser", DisplayName: dbtest.Ptr("Test User")}}}, nil
			},
			UpsertHashtagFn: func(ctx context.Context, text string) (db.Hashtag, error) {
				if text == "world" {
					hashtagCalled = true
				}
				return db.Hashtag{ID: 1, Text: text}, nil
			},
			LinkTweetHashtagFn: func(ctx context.Context, arg db.LinkTweetHashtagParams) error {
				if arg.TweetID == 10 && arg.HashtagID == 1 {
					linkCalled = true
				}
				return nil
			},
		}

		ucSuccess := tweet.New(tweet.Config{}, successStore, &storagetest.Fake{}, noopNotifier{})

		tweet, err := ucSuccess.Create(context.Background(), tweet.CreateInput{
			UserID:  1,
			Content: dbtest.Ptr("Hello #world"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tweet.ID != 10 {
			t.Fatalf("expected tweet ID 10, got %d", tweet.ID)
		}
		if !hashtagCalled || !linkCalled {
			t.Fatalf("expected hashtags to be extracted and linked, called=%v, link=%v", hashtagCalled, linkCalled)
		}
	})
}

func TestService_DeleteTweet(t *testing.T) {
	t.Run("wrong_owner", func(t *testing.T) {
		wrongStore := &dbtest.MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 2}}, nil // User 2 owns
			},
		}

		uc := tweet.New(tweet.Config{}, wrongStore, &storagetest.Fake{}, noopNotifier{})

		err := uc.Delete(context.Background(), 1, 10) // User 1 tries to delete
		if err == nil {
			t.Fatal("expected error for wrong owner deletion")
		}
		if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindForbidden {
			t.Fatalf("expected forbidden error, got %v", err)
		}
	})

	t.Run("success_with_media_delete", func(t *testing.T) {
		decrementCalled := false

		successStore := &dbtest.MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			ListMediaURLsInThreadFn: func(ctx context.Context, id int64) ([]*string, error) {
				return []*string{dbtest.Ptr(storagetest.BaseURL + "/tweets/1/img.png"), dbtest.Ptr("https://elsewhere.example/tweets/2/x.png")}, nil
			},
			ListHashtagUsageInThreadFn: func(ctx context.Context, id int64) ([]db.ListHashtagUsageInThreadRow, error) {
				return []db.ListHashtagUsageInThreadRow{{HashtagID: 5, Uses: 2}}, nil
			},
			DecrementHashtagUsageFn: func(ctx context.Context, arg db.DecrementHashtagUsageParams) error {
				if arg.ID == 5 && arg.Uses == 2 {
					decrementCalled = true
				}
				return nil
			},
			DeleteTweetByOwnerFn: func(ctx context.Context, arg db.DeleteTweetByOwnerParams) (db.Tweet, error) {
				return db.Tweet{ID: 10}, nil
			},
			DeleteUnusedHashtagFn: func(ctx context.Context, id int64) error {
				return nil
			},
		}

		storage := &storagetest.Fake{}
		uc := tweet.New(tweet.Config{}, successStore, storage, noopNotifier{})

		err := uc.Delete(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !decrementCalled {
			t.Fatal("expected hashtag decrement to be called")
		}
		// The hosted object is deleted; the external URL is never turned into a key.
		require.Equal(t, []string{"tweets/1/img.png"}, storage.Deleted)
	})
}

func TestService_Retweet(t *testing.T) {
	for name, created := range map[string]bool{"new_retweet_notifies": true, "repeat_retweet_is_a_noop": false} {
		t.Run(name, func(t *testing.T) {
			notified := false
			store := &dbtest.MockStore{
				GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
					return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}, IsRetweeted: true}, nil
				},
				CreateRetweetFn: func(ctx context.Context, arg db.CreateRetweetParams) (bool, error) {
					require.Equal(t, int64(10), arg.TweetID)
					return created, nil
				},
				CreateNotificationFn: func(ctx context.Context, arg db.CreateNotificationParams) (db.Notification, error) {
					notified = true
					return db.Notification{ID: 1}, nil
				},
				GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
					return []db.GetUsersByIDsRow{{User: db.User{ID: 1}}}, nil
				},
			}

			item, err := tweet.New(tweet.Config{}, store, &storagetest.Fake{}, noopNotifier{}).Retweet(context.Background(), 2, 10)

			require.NoError(t, err)
			require.Equal(t, int64(10), item.ID, "returns the retweeted tweet itself")
			require.True(t, item.IsRetweeted)
			require.Equal(t, created, notified)
		})
	}
}

func TestService_LikeUnlike(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	tweetID := int64(10)

	t.Run("like_success", func(t *testing.T) {
		called := false
		store := &dbtest.MockStore{
			LikeTweetFn: func(ctx context.Context, arg db.LikeTweetParams) (bool, error) {
				called = true
				return true, nil
			},
		}
		uc := tweet.New(tweet.Config{}, store, nil, noopNotifier{})
		err := uc.Like(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("unlike_success", func(t *testing.T) {
		called := false
		store := &dbtest.MockStore{
			UnlikeTweetFn: func(ctx context.Context, arg db.UnlikeTweetParams) (bool, error) {
				called = true
				return true, nil
			},
		}
		uc := tweet.New(tweet.Config{}, store, nil, noopNotifier{})
		err := uc.Unlike(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestService_UndoRetweet(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	tweetID := int64(10)

	t.Run("success", func(t *testing.T) {
		called := false
		store := &dbtest.MockStore{
			DeleteRetweetFn: func(ctx context.Context, arg db.DeleteRetweetParams) (bool, error) {
				called = true
				return true, nil
			},
		}
		uc := tweet.New(tweet.Config{}, store, nil, noopNotifier{})
		err := uc.UndoRetweet(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})
}
