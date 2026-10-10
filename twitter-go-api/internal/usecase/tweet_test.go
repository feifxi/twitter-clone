package usecase_test

import (
	"context"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type mockEmbeddingPublisher struct{}

func (m *mockEmbeddingPublisher) PublishEmbeddingEvent(ctx context.Context, tweetID int64, content string) error {
	return nil
}

func TestTweetUsecase_CreateTweet_Validation(t *testing.T) {
	uc := usecase.NewTweetUsecase(config.Config{}, &MockStore{}, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

	t.Run("empty_tweet", func(t *testing.T) {
		_, err := uc.CreateTweet(context.Background(), usecase.CreateTweetInput{
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
			_, err := uc.CreateTweet(context.Background(), usecase.CreateTweetInput{
				UserID:   1,
				MediaKey: ptr(key),
			})
			if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindBadRequest {
				t.Fatalf("expected bad request for %q, got %v", key, err)
			}
		})
	}

	t.Run("derives_media_type_from_owned_key", func(t *testing.T) {
		var created db.CreateTweetParams
		store := &MockStore{
			CreateTweetFn: func(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error) {
				created = arg
				return db.Tweet{ID: 10, UserID: arg.UserID, MediaUrl: arg.MediaUrl}, nil
			},
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 1}}}, nil
			},
		}
		uc := usecase.NewTweetUsecase(config.Config{}, store, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		_, err := uc.CreateTweet(context.Background(), usecase.CreateTweetInput{UserID: 1, MediaKey: ptr("tweets/1/clip.mp4")})
		require.NoError(t, err)
		require.Equal(t, usecase.MediaTypeVideo, *created.MediaType)
		require.Equal(t, fakeStorageBaseURL+"/tweets/1/clip.mp4", *created.MediaUrl)
	})

	t.Run("success_with_hashtags", func(t *testing.T) {
		hashtagCalled := false
		linkCalled := false

		successStore := &MockStore{
			CreateTweetFn: func(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error) {
				return db.Tweet{ID: 10, UserID: arg.UserID, Content: arg.Content}, nil
			},
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1, Content: ptr("Hello #world")}}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 1, Username: "testuser", DisplayName: ptr("Test User")}}}, nil
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

		ucSuccess := usecase.NewTweetUsecase(config.Config{EnableRAG: false}, successStore, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		tweet, err := ucSuccess.CreateTweet(context.Background(), usecase.CreateTweetInput{
			UserID:  1,
			Content: ptr("Hello #world"),
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

func TestTweetUsecase_DeleteTweet(t *testing.T) {
	t.Run("wrong_owner", func(t *testing.T) {
		wrongStore := &MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 2}}, nil // User 2 owns
			},
		}

		uc := usecase.NewTweetUsecase(config.Config{}, wrongStore, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		err := uc.DeleteTweet(context.Background(), 1, 10) // User 1 tries to delete
		if err == nil {
			t.Fatal("expected error for wrong owner deletion")
		}
		if kind, ok := apperr.KindOf(err); !ok || kind != apperr.KindForbidden {
			t.Fatalf("expected forbidden error, got %v", err)
		}
	})

	t.Run("success_with_media_delete", func(t *testing.T) {
		decrementCalled := false

		successStore := &MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			ListMediaUrlsInThreadFn: func(ctx context.Context, id int64) ([]*string, error) {
				return []*string{ptr(fakeStorageBaseURL + "/tweets/1/img.png"), ptr("https://elsewhere.example/tweets/2/x.png")}, nil
			},
			ListHashtagUsageToDecrementForDeleteRootFn: func(ctx context.Context, id int64) ([]db.ListHashtagUsageToDecrementForDeleteRootRow, error) {
				return []db.ListHashtagUsageToDecrementForDeleteRootRow{
					{HashtagID: 5, DecrementBy: 2},
				}, nil
			},
			DecrementHashtagUsageByFn: func(ctx context.Context, arg db.DecrementHashtagUsageByParams) error {
				if arg.ID == 5 && arg.UsageCount == 2 {
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

		storage := &fakeStorage{}
		uc := usecase.NewTweetUsecase(config.Config{}, successStore, storage, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		err := uc.DeleteTweet(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !decrementCalled {
			t.Fatal("expected hashtag decrement to be called")
		}
		// The hosted object is deleted; the external URL is never turned into a key.
		require.Equal(t, []string{"tweets/1/img.png"}, storage.deleted)
	})
}

func TestTweetUsecase_Retweet(t *testing.T) {
	t.Run("success_retweet", func(t *testing.T) {
		successStore := &MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			CreateRetweetFn: func(ctx context.Context, arg db.CreateRetweetParams) (db.CreateRetweetRow, error) {
				return db.CreateRetweetRow{ID: 20, UserID: 2, RetweetID: arg.RetweetID}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 2}}}, nil
			},
		}

		uc := usecase.NewTweetUsecase(config.Config{}, successStore, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		item, err := uc.Retweet(context.Background(), 2, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != 10 {
			t.Fatalf("expected retweet to return original tweet structure with ID 10, got %v", item.ID)
		}
	})

	t.Run("duplicate_retweet", func(t *testing.T) {
		// Mock pgx.ErrNoRows for ON CONFLICT DO NOTHING in CreateRetweet
		duplicateStore := &MockStore{
			GetTweetFn: func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
				return db.GetTweetRow{Tweet: db.Tweet{ID: 10, UserID: 1}}, nil
			},
			CreateRetweetFn: func(ctx context.Context, arg db.CreateRetweetParams) (db.CreateRetweetRow, error) {
				// Simulate ON CONFLICT DO NOTHING (if implementation expects ErrNoRows or similar)
				// Actually the usecase handles errors.
				return db.CreateRetweetRow{}, pgx.ErrNoRows
			},
			GetUserRetweetFn: func(ctx context.Context, arg db.GetUserRetweetParams) (db.Tweet, error) {
				return db.Tweet{ID: 20, UserID: 2, RetweetID: ptr(int64(10))}, nil
			},
			GetUsersByIDsFn: func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
				return []db.GetUsersByIDsRow{{User: db.User{ID: 2}}}, nil
			},
		}

		uc := usecase.NewTweetUsecase(config.Config{}, duplicateStore, &fakeStorage{}, &mockEmbeddingPublisher{}, func(n db.Notification) {})

		item, err := uc.Retweet(context.Background(), 2, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != 10 {
			t.Fatalf("expected retweet to return original tweet structure with ID 10, got %v", item.ID)
		}
	})
}

func TestTweetUsecase_LikeUnlike(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	tweetID := int64(10)

	t.Run("like_success", func(t *testing.T) {
		called := false
		store := &MockStore{
			LikeTweetFn: func(ctx context.Context, arg db.LikeTweetParams) (bool, error) {
				called = true
				return true, nil
			},
		}
		uc := usecase.NewTweetUsecase(config.Config{}, store, nil, nil, nil)
		err := uc.LikeTweet(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("unlike_success", func(t *testing.T) {
		called := false
		store := &MockStore{
			UnlikeTweetFn: func(ctx context.Context, arg db.UnlikeTweetParams) (bool, error) {
				called = true
				return true, nil
			},
		}
		uc := usecase.NewTweetUsecase(config.Config{}, store, nil, nil, nil)
		err := uc.UnlikeTweet(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})
}

func TestTweetUsecase_UndoRetweet(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	tweetID := int64(10)

	t.Run("success", func(t *testing.T) {
		called := false
		store := &MockStore{
			DeleteRetweetByUserFn: func(ctx context.Context, arg db.DeleteRetweetByUserParams) (db.DeleteRetweetByUserRow, error) {
				called = true
				return db.DeleteRetweetByUserRow{ID: 20, UserID: userID, RetweetID: ptr(tweetID)}, nil
			},
		}
		uc := usecase.NewTweetUsecase(config.Config{}, store, nil, nil, nil)
		err := uc.UndoRetweet(ctx, userID, tweetID)
		require.NoError(t, err)
		require.True(t, called)
	})
}
