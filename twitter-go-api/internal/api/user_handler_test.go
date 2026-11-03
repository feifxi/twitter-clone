package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
)

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
	ctx, rec := newHandlerTestContext(http.MethodPatch, "/api/v1/users/me", bytes.NewBufferString(reqBody), "application/json")
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

func TestUpdateProfileRejectsEmptyJSON(t *testing.T) {
	t.Parallel()

	// Sending valid JSON with no fields is fine (nothing to update)
	// but invalid JSON should fail
	reqBody := `not-json`
	ctx, rec := newHandlerTestContext(http.MethodPatch, "/api/v1/users/me", bytes.NewBufferString(reqBody), "application/json")
	setAuthorizedUser(ctx, 1)

	s := &Server{}
	s.updateProfile(ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProfileRequestValidation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body  string
		valid bool
	}{
		"nothing to change":     {`{}`, true},
		"clear the website":     {`{"website":""}`, true},
		"a website":             {`{"website":"https://example.com"}`, true},
		"a website without URL": {`{"website":"example"}`, false},
		"clear the bio":         {`{"bio":"","location":""}`, true},
		"blank name":            {`{"displayName":""}`, false},
		"a name":                {`{"displayName":"Chanom"}`, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, _ := newHandlerTestContext(http.MethodPatch, "/api/v1/users/me", bytes.NewBufferString(tc.body), "application/json")
			var req updateProfileRequest
			err := ctx.ShouldBindJSON(&req)
			if (err == nil) != tc.valid {
				t.Fatalf("valid = %v, want %v (err %v)", err == nil, tc.valid, err)
			}
		})
	}
}
