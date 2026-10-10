package httpapi

import (
	"context"
	"iter"

	"github.com/chanombude/twitter-go-api/internal/assistant"
	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/chanombude/twitter-go-api/internal/user"
)

// The interfaces below are what the handlers need from each feature package.
// They live here, at the consumer, so handlers can be tested with small fakes.

type AuthService interface {
	LoginWithGoogle(ctx context.Context, idToken string) (auth.Session, error)
	Refresh(ctx context.Context, refreshToken string) (auth.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID int64) (domain.User, error)
}

type UserService interface {
	Get(ctx context.Context, userID int64, viewerID *int64) (domain.User, error)
	GetByUsername(ctx context.Context, username string, viewerID *int64) (domain.User, error)
	CompleteOnboarding(ctx context.Context, userID int64, username, displayName string) (domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, in user.UpdateProfileInput) (domain.User, error)
	Follow(ctx context.Context, followerID, targetID int64) (bool, error)
	Unfollow(ctx context.Context, followerID, targetID int64) error
	Followers(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error)
	Following(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error)
}

type TweetService interface {
	Create(ctx context.Context, in tweet.CreateInput) (domain.Tweet, error)
	Delete(ctx context.Context, userID, tweetID int64) error
	Get(ctx context.Context, tweetID int64, viewerID *int64) (domain.Tweet, error)
	ListReplies(ctx context.Context, tweetID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Like(ctx context.Context, userID, tweetID int64) error
	Unlike(ctx context.Context, userID, tweetID int64) error
	Retweet(ctx context.Context, userID, tweetID int64) (domain.Tweet, error)
	UndoRetweet(ctx context.Context, userID, tweetID int64) error
}

type FeedService interface {
	Global(ctx context.Context, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Following(ctx context.Context, userID int64, req pagination.Request) (pagination.Page[domain.Tweet], error)
	Posts(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Replies(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Media(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Likes(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
}

type SearchService interface {
	Users(ctx context.Context, query string, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error)
	Tweets(ctx context.Context, query string, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error)
	Hashtags(ctx context.Context, query string, limit int32) ([]domain.Hashtag, error)
}

type DiscoveryService interface {
	TrendingHashtags(ctx context.Context, limit int32) ([]domain.Hashtag, error)
	SuggestedUsers(ctx context.Context, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error)
}

type NotificationService interface {
	List(ctx context.Context, userID int64, req pagination.Request) (pagination.Page[domain.Notification], error)
	CountUnread(ctx context.Context, userID int64) (int64, error)
	MarkAllRead(ctx context.Context, userID int64) error
}

type MessageService interface {
	Conversations(ctx context.Context, userID int64, req pagination.Request) (pagination.Page[domain.Conversation], error)
	Messages(ctx context.Context, userID, conversationID int64, req pagination.Request) (pagination.Page[domain.Message], error)
	SendToUser(ctx context.Context, senderID, recipientID int64, content string) (domain.Message, error)
	SendToConversation(ctx context.Context, senderID, conversationID int64, content string) (domain.Message, error)
}

type AssistantService interface {
	Chat(ctx context.Context, in assistant.Input) (iter.Seq2[string, error], error)
}

type MediaService interface {
	PresignUpload(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (media.Upload, error)
}

// Services bundles every feature the HTTP layer exposes.
type Services struct {
	Auth          AuthService
	Users         UserService
	Tweets        TweetService
	Feeds         FeedService
	Search        SearchService
	Discovery     DiscoveryService
	Notifications NotificationService
	Messages      MessageService
	Assistant     AssistantService
	Media         MediaService
}
