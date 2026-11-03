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

func TestCreateTweet_ServiceErrorReturnsInternalError(t *testing.T) {
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

func TestCreateTweetRejectsInvalidMediaKey(t *testing.T) {
	t.Parallel()

	reqBody := `{"content":"hello","mediaKey":"invalid-no-slash"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/tweets", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	mock := &fakeTweets{
		createFn: func(_ context.Context, _ tweet.CreateInput) (domain.Tweet, error) {
			return domain.Tweet{}, apperr.BadRequest("invalid media key")
		},
	}
	s := &Server{svc: Services{Tweets: mock}}
	s.createTweet(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid media key") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}
