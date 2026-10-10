package api

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/chanombude/twitter-go-api/internal/user"
)

// The fakes embed the service interface and override only what tests use;
// calling anything else panics, which flags an unexpected call.

type fakeTweets struct {
	TweetService
	createFn func(ctx context.Context, in tweet.CreateInput) (domain.Tweet, error)
}

func (f *fakeTweets) Create(ctx context.Context, in tweet.CreateInput) (domain.Tweet, error) {
	return f.createFn(ctx, in)
}

func (f *fakeTweets) Delete(context.Context, int64, int64) error { return nil }

type fakeUsers struct {
	UserService
	updateProfileFn func(ctx context.Context, userID int64, in user.UpdateProfileInput) (domain.User, error)
	followersFn     func(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error)
	followFn        func(ctx context.Context, followerID, targetID int64) (bool, error)
}

func (f *fakeUsers) UpdateProfile(ctx context.Context, userID int64, in user.UpdateProfileInput) (domain.User, error) {
	return f.updateProfileFn(ctx, userID, in)
}

func (f *fakeUsers) Followers(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error) {
	return f.followersFn(ctx, userID, req, viewerID)
}

func (f *fakeUsers) Follow(ctx context.Context, followerID, targetID int64) (bool, error) {
	if f.followFn == nil {
		return true, nil
	}
	return f.followFn(ctx, followerID, targetID)
}

type fakeFeeds struct {
	FeedService
	globalFn func(ctx context.Context, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
}

func (f *fakeFeeds) Global(ctx context.Context, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error) {
	return f.globalFn(ctx, req, viewerID)
}

type fakeAuth struct {
	AuthService
	refreshFn func(ctx context.Context, refreshToken string) (auth.Session, error)
	logoutFn  func(ctx context.Context, refreshToken string) error
}

func (f *fakeAuth) Refresh(ctx context.Context, refreshToken string) (auth.Session, error) {
	return f.refreshFn(ctx, refreshToken)
}

func (f *fakeAuth) Logout(ctx context.Context, refreshToken string) error {
	return f.logoutFn(ctx, refreshToken)
}

type fakeSearch struct {
	SearchService
	hashtagsFn func(ctx context.Context, query string, limit int32) ([]domain.Hashtag, error)
}

func (f *fakeSearch) Hashtags(ctx context.Context, query string, limit int32) ([]domain.Hashtag, error) {
	return f.hashtagsFn(ctx, query, limit)
}

type fakeMedia struct {
	presignFn func(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (media.Upload, error)
}

func (f *fakeMedia) PresignUpload(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (media.Upload, error) {
	return f.presignFn(ctx, userID, contentType, folder, contentLength)
}
