package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
)

type fakeTweets struct {
	createTweetFn func(ctx context.Context, input tweet.CreateInput) (domain.Tweet, error)
}

func (m *fakeTweets) Create(ctx context.Context, input tweet.CreateInput) (domain.Tweet, error) {
	return m.createTweetFn(ctx, input)
}
func (m *fakeTweets) Delete(context.Context, int64, int64) error { return nil }
func (m *fakeTweets) Get(context.Context, int64, *int64) (domain.Tweet, error) {
	return domain.Tweet{}, nil
}
func (m *fakeTweets) ListReplies(context.Context, int64, int32, int32, *int64) ([]domain.Tweet, error) {
	return nil, nil
}
func (m *fakeTweets) Like(context.Context, int64, int64) error   { return nil }
func (m *fakeTweets) Unlike(context.Context, int64, int64) error { return nil }
func (m *fakeTweets) Retweet(context.Context, int64, int64) (domain.Tweet, error) {
	return domain.Tweet{}, nil
}
func (m *fakeTweets) UndoRetweet(context.Context, int64, int64) error { return nil }

type fakeUsers struct {
	updateProfileFn func(ctx context.Context, userID int64, input user.UpdateProfileInput) (domain.User, error)
	listFollowersFn func(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]domain.User, error)
	followUserFn    func(ctx context.Context, followerID, targetUserID int64) (bool, error)
}

func (m *fakeUsers) Get(context.Context, int64, *int64) (domain.User, error) {
	return domain.User{}, nil
}
func (m *fakeUsers) UpdateProfile(ctx context.Context, userID int64, input user.UpdateProfileInput) (domain.User, error) {
	return m.updateProfileFn(ctx, userID, input)
}
func (m *fakeUsers) Follow(ctx context.Context, followerID, targetUserID int64) (bool, error) {
	if m.followUserFn != nil {
		return m.followUserFn(ctx, followerID, targetUserID)
	}
	return true, nil
}
func (m *fakeUsers) Unfollow(context.Context, int64, int64) error { return nil }
func (m *fakeUsers) Followers(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]domain.User, error) {
	if m.listFollowersFn != nil {
		return m.listFollowersFn(ctx, targetUserID, page, size, viewerID)
	}
	return nil, nil
}
func (m *fakeUsers) Following(context.Context, int64, int32, int32, *int64) ([]domain.User, error) {
	return nil, nil
}

type fakeFeeds struct {
	getGlobalFeedFn func(ctx context.Context, page, size int32, viewerID *int64) ([]domain.Tweet, error)
}

func (m *fakeFeeds) Global(ctx context.Context, page, size int32, viewerID *int64) ([]domain.Tweet, error) {
	return m.getGlobalFeedFn(ctx, page, size, viewerID)
}
func (m *fakeFeeds) Following(context.Context, int64, int32, int32) ([]domain.Tweet, error) {
	return nil, nil
}
func (m *fakeFeeds) ForUser(context.Context, int64, int32, int32, *int64) ([]domain.Tweet, error) {
	return nil, nil
}

type fakeAuth struct {
	refreshSessionFn func(ctx context.Context, refreshToken string) (auth.Session, error)
	logoutFn         func(ctx context.Context, userID *int64, refreshToken *string)
}

func (m *fakeAuth) LoginWithGoogle(context.Context, string) (auth.Session, error) {
	return auth.Session{}, nil
}
func (m *fakeAuth) Refresh(ctx context.Context, refreshToken string) (auth.Session, error) {
	return m.refreshSessionFn(ctx, refreshToken)
}
func (m *fakeAuth) Logout(ctx context.Context, userID *int64, refreshToken *string) {
	if m.logoutFn != nil {
		m.logoutFn(ctx, userID, refreshToken)
	}
}
func (m *fakeAuth) Me(context.Context, int64) (domain.User, error) {
	return domain.User{}, nil
}

func TestCreateTweetSuccessPassesMediaKey(t *testing.T) {
	t.Parallel()

	var gotMediaKey string
	called := false
	mock := &fakeTweets{
		createTweetFn: func(_ context.Context, input tweet.CreateInput) (domain.Tweet, error) {
			called = true
			if input.MediaKey == nil {
				t.Fatal("expected media key input")
			}
			gotMediaKey = *input.MediaKey
			return domain.Tweet{
				ID:        123,
				Content:   input.Content,
				CreatedAt: time.Now(),
				Author: domain.User{
					ID:       input.UserID,
					Username: "tester",
					Email:    "tester@example.com",
				},
			}, nil
		},
	}

	reqBody := `{"content":"hello","mediaKey":"tweets/7/photo.png"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/tweets", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 7)

	s := &Server{svc: Services{Tweets: mock}}
	s.createTweet(ctx)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("expected createTweet service to be called")
	}
	if gotMediaKey != "tweets/7/photo.png" {
		t.Fatalf("expected tweets/7/photo.png, got %q", gotMediaKey)
	}
	if !strings.Contains(rec.Body.String(), `"id":123`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestUpdateProfileSuccessPassesAvatarKey(t *testing.T) {
	t.Parallel()

	var gotAvatarKey string
	called := false
	mock := &fakeUsers{
		updateProfileFn: func(_ context.Context, userID int64, input user.UpdateProfileInput) (domain.User, error) {
			called = true
			if userID != 5 {
				t.Fatalf("expected userID=5, got %d", userID)
			}
			if input.AvatarKey == nil {
				t.Fatal("expected avatar key input")
			}
			gotAvatarKey = *input.AvatarKey
			return domain.User{
				ID:       userID,
				Username: "tester",
				Email:    "tester@example.com",
			}, nil
		},
	}

	reqBody := `{"displayName":"new name","avatarKey":"avatars/uuid_avatar.png"}`
	ctx, rec := newHandlerTestContext(http.MethodPut, "/api/v1/users/profile", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 5)

	s := &Server{svc: Services{Users: mock}}
	s.updateProfile(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if !called {
		t.Fatal("expected updateProfile service to be called")
	}
	if gotAvatarKey != "avatars/uuid_avatar.png" {
		t.Fatalf("expected avatars/uuid_avatar.png, got %q", gotAvatarKey)
	}
}

func TestRefreshTokenSuccessSetsTokensInResponse(t *testing.T) {
	t.Parallel()

	mock := &fakeAuth{
		refreshSessionFn: func(_ context.Context, refreshToken string) (auth.Session, error) {
			if refreshToken != "r-old" {
				t.Fatalf("unexpected refresh token: %s", refreshToken)
			}
			return auth.Session{
				AccessToken:  "a-new",
				RefreshToken: "r-new",
				User: domain.User{
					ID:       9,
					Username: "tester",
					Email:    "tester@example.com",
				},
			}, nil
		},
	}

	reqBody := `{"refreshToken":"r-old"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(reqBody), "application/json")

	s := &Server{svc: Services{Auth: mock}}
	s.refreshToken(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var got AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if got.AccessToken != "a-new" || got.RefreshToken != "r-new" {
		t.Fatalf("expected both tokens in response, got: %+v", got)
	}
}

func TestLogoutWithAuthPayloadCallsUsecaseWithUserID(t *testing.T) {
	t.Parallel()

	called := false
	mock := &fakeAuth{
		logoutFn: func(_ context.Context, userID *int64, refreshToken *string) {
			called = true
			if userID == nil || *userID != 77 {
				t.Fatalf("expected userID=77, got %+v", userID)
			}
			if refreshToken != nil {
				t.Fatalf("expected nil refresh token, got %v", *refreshToken)
			}
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/logout", nil, "")
	setAuthorizedUser(ctx, 77)

	s := &Server{svc: Services{Auth: mock}}
	s.logout(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Fatal("expected logout service to be called")
	}
}

func TestGetGlobalFeedReturnsItemsAndHasNext(t *testing.T) {
	t.Parallel()

	var gotSize int32
	mock := &fakeFeeds{
		getGlobalFeedFn: func(_ context.Context, _ int32, size int32, _ *int64) ([]domain.Tweet, error) {
			gotSize = size
			now := time.Now()
			return []domain.Tweet{
				{ID: 1, CreatedAt: now, Author: domain.User{ID: 11, Username: "u1", Email: "u1@example.com"}},
				{ID: 2, CreatedAt: now, Author: domain.User{ID: 12, Username: "u2", Email: "u2@example.com"}},
				{ID: 3, CreatedAt: now, Author: domain.User{ID: 13, Username: "u3", Email: "u3@example.com"}},
			}, nil
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global?size=2", nil, "")
	s := &Server{svc: Services{Feeds: mock}}
	s.getGlobalFeed(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotSize != 3 {
		t.Fatalf("expected service size=3 (size+1), got %d", gotSize)
	}

	var got struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		HasNext    bool    `json:"hasNext"`
		NextCursor *string `json:"nextCursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items after trim, got %d", len(got.Items))
	}
	if !got.HasNext {
		t.Fatal("expected hasNext=true")
	}
	if got.NextCursor == nil || *got.NextCursor == "" {
		t.Fatal("expected nextCursor to be set")
	}
	if got.Items[0].ID != 1 || got.Items[1].ID != 2 {
		t.Fatalf("unexpected trimmed items: %+v", got.Items)
	}
}

func TestListFollowersReturnsItemsAndHasNext(t *testing.T) {
	t.Parallel()

	var gotSize int32
	mock := &fakeUsers{
		listFollowersFn: func(_ context.Context, targetUserID int64, _ int32, size int32, _ *int64) ([]domain.User, error) {
			gotSize = size
			if targetUserID != 99 {
				t.Fatalf("expected targetUserID=99, got %d", targetUserID)
			}
			return []domain.User{
				{ID: 1, Username: "a", Email: "a@example.com"},
				{ID: 2, Username: "b", Email: "b@example.com"},
				{ID: 3, Username: "c", Email: "c@example.com"},
			}, nil
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/users/99/followers?size=2", nil, "")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}
	s := &Server{svc: Services{Users: mock}}
	s.listFollowers(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if gotSize != 3 {
		t.Fatalf("expected service size=3 (size+1), got %d", gotSize)
	}

	var got struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		HasNext    bool    `json:"hasNext"`
		NextCursor *string `json:"nextCursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items after trim, got %d", len(got.Items))
	}
	if !got.HasNext {
		t.Fatal("expected hasNext=true")
	}
	if got.NextCursor == nil || *got.NextCursor == "" {
		t.Fatal("expected nextCursor to be set")
	}
}
