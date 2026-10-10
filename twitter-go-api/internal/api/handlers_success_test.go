package api

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
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
)

func TestCreateTweetSuccessPassesMediaKey(t *testing.T) {
	t.Parallel()

	var gotMediaKey string
	called := false
	mock := &fakeTweets{
		createFn: func(_ context.Context, input tweet.CreateInput) (domain.Tweet, error) {
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
		refreshFn: func(_ context.Context, refreshToken string) (auth.Session, error) {
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

func TestLogoutEndsTheSessionOfTheGivenRefreshToken(t *testing.T) {
	t.Parallel()

	var got string
	fake := &fakeAuth{logoutFn: func(_ context.Context, refreshToken string) error {
		got = refreshToken
		return nil
	}}

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(`{"refreshToken":"r-1"}`), "application/json")
	(&Server{svc: Services{Auth: fake}}).logout(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got != "r-1" {
		t.Fatalf("expected the body's refresh token to be revoked, got %q", got)
	}
}

func TestGetGlobalFeedPassesPageRequestAndEncodesNextCursor(t *testing.T) {
	t.Parallel()

	var got pagination.Request
	fake := &fakeFeeds{globalFn: func(_ context.Context, req pagination.Request, _ *int64) (pagination.Page[domain.Tweet], error) {
		got = req
		return pagination.Page[domain.Tweet]{
			Items: []domain.Tweet{{ID: 1, CreatedAt: time.Now(), Author: domain.User{ID: 11, Email: "u1@example.com"}}},
			Next:  &pagination.Cursor{ID: 1},
		}, nil
	}}

	cursor := pagination.Encode(pagination.Cursor{ID: 5})
	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global?size=2&cursor="+cursor, nil, "")
	(&Server{svc: Services{Feeds: fake}}).getGlobalFeed(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if got.Limit != 2 || got.After == nil || got.After.ID != 5 {
		t.Fatalf("unexpected page request: %+v", got)
	}

	var body PageResponse[TweetResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(body.Items) != 1 || !body.HasNext || body.NextCursor == nil {
		t.Fatalf("unexpected page: %+v", body)
	}
	if next, err := pagination.Decode(*body.NextCursor); err != nil || next.ID != 1 {
		t.Fatalf("next cursor does not round-trip: %v %+v", err, next)
	}
	if strings.Contains(rec.Body.String(), "u1@example.com") {
		t.Fatal("feed must not leak author emails")
	}
}

func TestPageRequestValidation(t *testing.T) {
	t.Parallel()

	for _, query := range []string{"?cursor=not-a-cursor", "?size=0", "?size=abc"} {
		ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global"+query, nil, "")
		(&Server{svc: Services{Feeds: &fakeFeeds{}}}).getGlobalFeed(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", query, rec.Code)
		}
	}

	ctx, _ := newHandlerTestContext(http.MethodGet, "/x?size=500", nil, "")
	req, ok := parsePageRequest(ctx)
	if !ok || req.Limit != pagination.MaxLimit {
		t.Fatalf("expected size to be capped at %d, got %+v", pagination.MaxLimit, req)
	}
}

func TestListFollowersUsesPathIDAndPage(t *testing.T) {
	t.Parallel()

	var gotID int64
	fake := &fakeUsers{followersFn: func(_ context.Context, userID int64, req pagination.Request, _ *int64) (pagination.Page[domain.User], error) {
		gotID = userID
		return pagination.Page[domain.User]{Items: []domain.User{{ID: 1, Username: "a"}}}, nil
	}}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/users/99/followers", nil, "")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}
	(&Server{svc: Services{Users: fake}}).listFollowers(ctx)

	if rec.Code != http.StatusOK || gotID != 99 {
		t.Fatalf("expected 200 for user 99, got %d (id %d)", rec.Code, gotID)
	}
	var body PageResponse[UserResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Items) != 1 || body.HasNext {
		t.Fatalf("unexpected page: %+v (%v)", body, err)
	}
}
