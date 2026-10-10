package usecase

import (
	"context"
	"iter"

	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/service"
	"github.com/chanombude/twitter-go-api/internal/token"
	"google.golang.org/api/idtoken"
	"google.golang.org/genai"
)

// --- Service Interfaces ---

type AuthService interface {
	LoginWithGoogle(ctx context.Context, idToken string) (AuthResult, error)
	RefreshSession(ctx context.Context, refreshToken string) (AuthResult, error)
	Logout(ctx context.Context, userID *int64, refreshToken *string)
	GetMe(ctx context.Context, userID int64) (UserItem, error)
}

type UserService interface {
	GetUser(ctx context.Context, targetUserID int64, viewerID *int64) (UserItem, error)
	UpdateProfile(ctx context.Context, userID int64, input UpdateProfileInput) (UserItem, error)
	FollowUser(ctx context.Context, followerID, targetUserID int64) (bool, error)
	UnfollowUser(ctx context.Context, followerID, targetUserID int64) error
	ListFollowers(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]UserItem, error)
	ListFollowing(ctx context.Context, targetUserID int64, page, size int32, viewerID *int64) ([]UserItem, error)
}

type TweetService interface {
	CreateTweet(ctx context.Context, input CreateTweetInput) (TweetItem, error)
	DeleteTweet(ctx context.Context, userID, tweetID int64) error
	GetTweet(ctx context.Context, tweetID int64, viewerID *int64) (TweetItem, error)
	ListReplies(ctx context.Context, tweetID int64, page, size int32, viewerID *int64) ([]TweetItem, error)
	LikeTweet(ctx context.Context, userID, tweetID int64) error
	UnlikeTweet(ctx context.Context, userID, tweetID int64) error
	Retweet(ctx context.Context, userID, tweetID int64) (TweetItem, error)
	UndoRetweet(ctx context.Context, userID, tweetID int64) error
}

type FeedService interface {
	GetGlobalFeed(ctx context.Context, page, size int32, viewerID *int64) ([]TweetItem, error)
	GetFollowingFeed(ctx context.Context, userID int64, page, size int32) ([]TweetItem, error)
	GetUserFeed(ctx context.Context, userID int64, page, size int32, viewerID *int64) ([]TweetItem, error)
}

type SearchService interface {
	SearchUsers(ctx context.Context, query string, page, size int32, viewerID *int64) ([]UserItem, error)
	SearchTweets(ctx context.Context, query string, page, size int32, viewerID *int64) ([]TweetItem, error)
	SearchHashtags(ctx context.Context, query string, limit int32) ([]db.Hashtag, error)
}

type DiscoveryService interface {
	GetTrendingHashtags(ctx context.Context, limit int32) ([]db.Hashtag, error)
	GetSuggestedUsers(ctx context.Context, page, size int32, viewerID *int64) ([]UserItem, error)
}

type NotificationService interface {
	ListNotifications(ctx context.Context, userID int64, page, size int32) ([]NotificationItem, error)
	CountUnreadNotifications(ctx context.Context, userID int64) (int64, error)
	MarkAllNotificationsRead(ctx context.Context, userID int64) error
	HydrateNotification(ctx context.Context, notification db.Notification) (NotificationItem, error)
}

type MessageService interface {
	ListConversations(ctx context.Context, userID int64, page, size int32) ([]ConversationItem, error)
	ListMessages(ctx context.Context, userID, conversationID int64, page, size int32) ([]MessageItem, error)
	SendMessageToUser(ctx context.Context, senderID, recipientID int64, content string) (MessageItem, []int64, error)
	SendMessageToConversation(ctx context.Context, senderID, conversationID int64, content string) (MessageItem, []int64, error)
}

type AssistantService interface {
	Chat(ctx context.Context, input AssistantInput) (iter.Seq2[string, error], error)
}

type UploadService interface {
	GeneratePresignedURL(ctx context.Context, userID int64, contentType, folder string, contentLength int64) (PresignedUpload, error)
}

// --- Domain Helper Interfaces ---

type GoogleVerifier interface {
	Validate(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error)
}

// --- Implementation Structs ---

type AuthUsecase struct {
	config         config.Config
	store          db.Store
	tokenMaker     token.Maker
	googleVerifier GoogleVerifier
}

type UserUsecase struct {
	store               db.Store
	storage             service.StorageService
	publishNotification func(db.Notification)
}

type TweetUsecase struct {
	config              config.Config
	store               db.Store
	storage             service.StorageService
	embeddingPublisher  service.EmbeddingPublisher
	publishNotification func(db.Notification)
}

type FeedUsecase struct {
	store db.Store
}

type SearchUsecase struct {
	store db.Store
}

type DiscoveryUsecase struct {
	store db.Store
}

type NotificationUsecase struct {
	store db.Store
}

type MessageUsecase struct {
	store db.Store
}

type AssistantUsecase struct {
	config config.Config
	store  db.Store
	genai  *genai.Client
}

type UploadUsecase struct {
	config  config.Config
	storage service.StorageService
}

// --- Interface Implementation Checks ---

var (
	_ AuthService         = (*AuthUsecase)(nil)
	_ UserService         = (*UserUsecase)(nil)
	_ TweetService        = (*TweetUsecase)(nil)
	_ FeedService         = (*FeedUsecase)(nil)
	_ SearchService       = (*SearchUsecase)(nil)
	_ DiscoveryService    = (*DiscoveryUsecase)(nil)
	_ NotificationService = (*NotificationUsecase)(nil)
	_ MessageService      = (*MessageUsecase)(nil)
	_ AssistantService    = (*AssistantUsecase)(nil)
	_ UploadService       = (*UploadUsecase)(nil)
)

// --- Constructors ---

func NewAuthUsecase(cfg config.Config, store db.Store, tokenMaker token.Maker, googleVerifier GoogleVerifier) *AuthUsecase {
	return &AuthUsecase{
		config:         cfg,
		store:          store,
		tokenMaker:     tokenMaker,
		googleVerifier: googleVerifier,
	}
}

func NewUserUsecase(store db.Store, storage service.StorageService, publishNotification func(db.Notification)) *UserUsecase {
	return &UserUsecase{
		store:               store,
		storage:             storage,
		publishNotification: publishNotification,
	}
}

func NewTweetUsecase(cfg config.Config, store db.Store, storage service.StorageService, embeddingPublisher service.EmbeddingPublisher, publishNotification func(db.Notification)) *TweetUsecase {
	return &TweetUsecase{
		config:              cfg,
		store:               store,
		storage:             storage,
		embeddingPublisher:  embeddingPublisher,
		publishNotification: publishNotification,
	}
}

func NewFeedUsecase(store db.Store) *FeedUsecase {
	return &FeedUsecase{store: store}
}

func NewSearchUsecase(store db.Store) *SearchUsecase {
	return &SearchUsecase{store: store}
}

func NewDiscoveryUsecase(store db.Store) *DiscoveryUsecase {
	return &DiscoveryUsecase{store: store}
}

func NewNotificationUsecase(store db.Store) *NotificationUsecase {
	return &NotificationUsecase{store: store}
}

func NewMessageUsecase(store db.Store) *MessageUsecase {
	return &MessageUsecase{store: store}
}

func NewAssistantUsecase(cfg config.Config, store db.Store, genaiClient *genai.Client) *AssistantUsecase {
	return &AssistantUsecase{
		config: cfg,
		store:  store,
		genai:  genaiClient,
	}
}

func NewUploadUsecase(cfg config.Config, storage service.StorageService) *UploadUsecase {
	return &UploadUsecase{
		config:  cfg,
		storage: storage,
	}
}

// --- Assistant Models ---

type AssistantInput struct {
	Query   string                 `json:"query" binding:"required"`
	History []AssistantHistoryItem `json:"history"`
}

type AssistantHistoryItem struct {
	Role string `json:"role"` // "user" or "model" (for Gemini)
	Text string `json:"text"`
}
