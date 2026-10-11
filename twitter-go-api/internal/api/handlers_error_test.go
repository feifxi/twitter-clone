package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type fakeTweetsNotFound struct{ TweetService }

func (m *fakeTweetsNotFound) Get(_ context.Context, _ int64, _ *int64) (domain.Tweet, error) {
	return domain.Tweet{}, pgx.ErrNoRows
}

func TestGetTweet_NotFoundReturns404(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/tweets/99", nil, "")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "99"}}

	s := &Server{svc: Services{Tweets: &fakeTweetsNotFound{}}}
	s.getTweet(ctx)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing tweet, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["code"] != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND code, got %v", out["code"])
	}
}

func TestCreateTweet_UsecaseErrorReturnsInternalError(t *testing.T) {
	t.Parallel()

	mock := &fakeTweets{
		createFn: func(_ context.Context, _ tweet.CreateInput) (domain.Tweet, error) {
			return domain.Tweet{}, apperr.Internal("db failure", errors.New("connection reset"))
		},
	}

	reqBody := `{"content":"hello"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/tweets", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	s := &Server{svc: Services{Tweets: mock}}
	s.createTweet(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for service error, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["code"] != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR code, got %v", out["code"])
	}
}

func TestCreateTweetWithoutMedia_SuccessPath(t *testing.T) {
	t.Parallel()

	mock := &fakeTweets{
		createFn: func(_ context.Context, input tweet.CreateInput) (domain.Tweet, error) {
			content := "hello"
			return domain.Tweet{
				ID:        77,
				Content:   &content,
				CreatedAt: time.Now(),
				Author:    domain.User{ID: input.UserID, Username: "tester", Email: "t@t.com"},
			}, nil
		},
	}

	reqBody := `{"content":"hello"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/tweets", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 3)

	s := &Server{svc: Services{Tweets: mock}}
	s.createTweet(ctx)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["id"] != float64(77) {
		t.Fatalf("expected id=77, got %v", out["id"])
	}
}

func TestFollowUser_CannotFollowSelf(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/users/5/follow", nil, "")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "5"}}
	setAuthorizedUser(ctx, 5)

	mock := &fakeUsers{
		followFn: func(ctx context.Context, followerID, targetUserID int64) (bool, error) {
			if followerID == targetUserID {
				return false, apperr.BadRequest("cannot follow yourself")
			}
			return true, nil
		},
	}
	s := &Server{svc: Services{Users: mock}}
	s.followUser(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for self-follow, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cannot follow yourself") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestDeleteTweet_MissingAuth(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodDelete, "/api/v1/tweets/1", nil, "")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	s := &Server{svc: Services{Tweets: &fakeTweets{}}}
	s.deleteTweet(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated delete, got %d", rec.Code)
	}
}

func TestGetUnreadNotificationCount_MissingAuth(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/notifications/unread-count", nil, "")

	s := &Server{}
	s.getUnreadNotificationCount(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMarkNotificationRead_MissingAuth(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/notifications/mark-read", nil, "")

	s := &Server{}
	s.markNotificationRead(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetGlobalFeed_UsecaseErrorPropagates(t *testing.T) {
	t.Parallel()

	mock := &fakeFeeds{
		globalFn: func(context.Context, pagination.Request, *int64) (pagination.Page[domain.FeedItem], error) {
			return pagination.Page[domain.FeedItem]{}, apperr.Internal("feed failure", errors.New("db error"))
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global", nil, "")
	s := &Server{svc: Services{Feeds: mock}}
	s.getGlobalFeed(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for feed error, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSearchHashtags_ReturnsResults(t *testing.T) {
	t.Parallel()

	mock := &fakeSearch{
		hashtagsFn: func(_ context.Context, _ string, _ int32) ([]domain.Hashtag, error) {
			return []domain.Hashtag{{Text: "golang"}}, nil
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/search/hashtags?q=go", nil, "")
	s := &Server{svc: Services{Search: mock}}
	s.searchHashtags(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v body=%s", err, rec.Body.String())
	}
	if len(out) != 1 || out[0]["text"] != "golang" {
		t.Fatalf("unexpected response: %v", out)
	}
}
