package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

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

func TestGetGlobalFeedPassesPageRequestAndEncodesNextCursor(t *testing.T) {
	t.Parallel()

	var got pagination.Request
	fake := &fakeFeeds{globalFn: func(_ context.Context, req pagination.Request, _ *int64) (pagination.Page[domain.FeedItem], error) {
		got = req
		return pagination.Page[domain.FeedItem]{
			Items: []domain.FeedItem{{Tweet: domain.Tweet{ID: 1, CreatedAt: time.Now(), Author: domain.User{ID: 11, Email: "u1@example.com"}}}},
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

	var body PageResponse[FeedItemResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(body.Items) != 1 || !body.HasNext || body.NextCursor == nil || body.Items[0].Kind != "tweet" || body.Items[0].Tweet.ID != 1 {
		t.Fatalf("unexpected page: %+v", body)
	}
	if next, err := pagination.Decode(*body.NextCursor); err != nil || next.ID != 1 {
		t.Fatalf("next cursor does not round-trip: %v %+v", err, next)
	}
	if strings.Contains(rec.Body.String(), "u1@example.com") {
		t.Fatal("feed must not leak author emails")
	}
}

// The feed item contract: every field present (nulls included), retweets wrap the
// original tweet, and the last page says nextCursor: null.
func TestFeedItemsAndLastPageShape(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC)
	fake := &fakeFeeds{globalFn: func(context.Context, pagination.Request, *int64) (pagination.Page[domain.FeedItem], error) {
		return pagination.Page[domain.FeedItem]{Items: []domain.FeedItem{
			{Tweet: domain.Tweet{ID: 1, CreatedAt: at, Author: domain.User{ID: 11, Username: "author"}}},
			{Tweet: domain.Tweet{ID: 2, CreatedAt: at, Author: domain.User{ID: 11, Username: "author"}},
				Retweet: &domain.Retweet{ID: 7, By: domain.User{ID: 12, Username: "fan"}, CreatedAt: at}},
		}}, nil
	}}
	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global", nil, "")
	(&Server{svc: Services{Feeds: fake}}).getGlobalFeed(ctx)

	var body struct {
		Items      []map[string]json.RawMessage `json:"items"`
		HasNext    bool                         `json:"hasNext"`
		NextCursor *string                      `json:"nextCursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !strings.Contains(rec.Body.String(), `"nextCursor":null`) || body.HasNext {
		t.Fatalf("the last page must say nextCursor: null, got %s", rec.Body.String())
	}
	if string(body.Items[0]["kind"]) != `"tweet"` || string(body.Items[0]["retweet"]) != "null" {
		t.Fatalf("plain tweet item: %s", rec.Body.String())
	}
	var rt RetweetResponse
	if string(body.Items[1]["kind"]) != `"retweet"` || json.Unmarshal(body.Items[1]["retweet"], &rt) != nil ||
		rt.ID != 7 || rt.Retweeter.Username != "fan" || !rt.CreatedAt.Equal(at) {
		t.Fatalf("retweet item: %s", rec.Body.String())
	}
	if strings.Contains(string(body.Items[1]["tweet"]), "retweetedBy") {
		t.Fatal("the tweet inside an item is the plain tweet shape")
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
