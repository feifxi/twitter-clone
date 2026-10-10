package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/chanombude/twitter-go-api/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newHandlerTestContext(method, path string, body *bytes.Buffer, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	reqBody := bytes.NewReader(nil)
	if body != nil {
		reqBody = bytes.NewReader(body.Bytes())
	}
	req := httptest.NewRequest(method, path, reqBody)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	return ctx, w
}

func setAuthorizedUser(ctx *gin.Context, userID int64) {
	ctx.Set(middleware.AuthorizationPayloadKey, &token.Payload{
		ID:        uuid.New(),
		UserID:    userID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	})
}

func TestCreateTweetRejectsInvalidMediaKey(t *testing.T) {
	t.Parallel()

	reqBody := `{"content":"hello","mediaKey":"invalid-no-slash"}`
	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/tweets", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	mock := &mockTweetUC{
		createTweetFn: func(_ context.Context, _ usecase.CreateTweetInput) (usecase.TweetItem, error) {
			return usecase.TweetItem{}, apperr.BadRequest("invalid media key")
		},
	}
	s := &Server{config: config.Config{}, tweetUC: mock}
	s.createTweet(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid media key") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestUpdateProfileRejectsEmptyJSON(t *testing.T) {
	t.Parallel()

	// Sending valid JSON with no fields is fine (nothing to update)
	// but invalid JSON should fail
	reqBody := `not-json`
	ctx, rec := newHandlerTestContext(http.MethodPut, "/api/v1/users/profile", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	s := &Server{config: config.Config{}}
	s.updateProfile(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRefreshTokenMissingToken(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/refresh", nil, "")

	s := &Server{}
	s.refreshToken(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if got["code"] != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED code, got %v", got["code"])
	}
}

func TestLogoutWithoutSessionStillSucceeds(t *testing.T) {
	t.Parallel()

	ctx, rec := newHandlerTestContext(http.MethodPost, "/api/v1/auth/logout", nil, "")

	s := &Server{}
	s.logout(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}
