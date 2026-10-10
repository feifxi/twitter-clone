// Package dbtest provides a hook-based fake of db.Store for unit tests.
package dbtest

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/db"
)

// MockStore is the unified mock for db.Store.
// It uses function hooks to allow individual tests to override behavior.
// Methods return zero values/nil if hooks are not provided.
type MockStore struct {
	// User
	CreateUserFn         func(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserFn            func(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error)
	GetUserByEmailFn     func(ctx context.Context, email string) (db.User, error)
	GetUserByUsernameFn  func(ctx context.Context, username string) (db.User, error)
	UpdateUserProfileFn  func(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error)
	FollowUserFn         func(ctx context.Context, arg db.FollowUserParams) (bool, error)
	UnfollowUserFn       func(ctx context.Context, arg db.UnfollowUserParams) (bool, error)
	ListFollowersUsersFn func(ctx context.Context, arg db.ListFollowersUsersParams) ([]db.ListFollowersUsersRow, error)
	ListFollowingUsersFn func(ctx context.Context, arg db.ListFollowingUsersParams) ([]db.ListFollowingUsersRow, error)
	GetFollowedUserIDsFn func(ctx context.Context, arg db.GetFollowedUserIDsParams) ([]int64, error)
	IsFollowingFn        func(ctx context.Context, arg db.IsFollowingParams) (bool, error)

	// Tweet
	CreateTweetFn               func(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error)
	GetTweetFn                  func(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error)
	ListTweetRepliesFn          func(ctx context.Context, arg db.ListTweetRepliesParams) ([]db.ListTweetRepliesRow, error)
	DeleteTweetByOwnerFn        func(ctx context.Context, arg db.DeleteTweetByOwnerParams) (db.Tweet, error)
	LikeTweetFn                 func(ctx context.Context, arg db.LikeTweetParams) (bool, error)
	UnlikeTweetFn               func(ctx context.Context, arg db.UnlikeTweetParams) (bool, error)
	IsTweetLikedFn              func(ctx context.Context, arg db.IsTweetLikedParams) (bool, error)
	IncrementParentReplyCountFn func(ctx context.Context, id int64) error
	DecrementParentReplyCountFn func(ctx context.Context, id int64) error
	CreateRetweetFn             func(ctx context.Context, arg db.CreateRetweetParams) (db.CreateRetweetRow, error)
	GetUserRetweetFn            func(ctx context.Context, arg db.GetUserRetweetParams) (db.Tweet, error)
	DeleteRetweetByUserFn       func(ctx context.Context, arg db.DeleteRetweetByUserParams) (db.DeleteRetweetByUserRow, error)
	ListMediaURLsInThreadFn     func(ctx context.Context, id int64) ([]*string, error)

	// Hashtag
	UpsertHashtagFn                            func(ctx context.Context, text string) (db.Hashtag, error)
	LinkTweetHashtagFn                         func(ctx context.Context, arg db.LinkTweetHashtagParams) error
	SearchHashtagsByPrefixFn                   func(ctx context.Context, arg db.SearchHashtagsByPrefixParams) ([]db.Hashtag, error)
	GetTrendingHashtagsLast24hFn               func(ctx context.Context, limit int32) ([]db.Hashtag, error)
	GetTopHashtagsAllTimeFn                    func(ctx context.Context, limit int32) ([]db.Hashtag, error)
	ListHashtagUsageToDecrementForDeleteRootFn func(ctx context.Context, id int64) ([]db.ListHashtagUsageToDecrementForDeleteRootRow, error)
	DecrementHashtagUsageByFn                  func(ctx context.Context, arg db.DecrementHashtagUsageByParams) error
	DeleteUnusedHashtagFn                      func(ctx context.Context, id int64) error

	// Auth
	CreateRefreshTokenFn        func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetRefreshTokenFn           func(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	DeleteRefreshTokenFn        func(ctx context.Context, tokenHash string) error
	DeleteRefreshTokensByUserFn func(ctx context.Context, userID int64) error

	// Feed
	ListForYouFeedFn    func(ctx context.Context, arg db.ListForYouFeedParams) ([]db.ListForYouFeedRow, error)
	ListFollowingFeedFn func(ctx context.Context, arg db.ListFollowingFeedParams) ([]db.ListFollowingFeedRow, error)
	ListUserTweetsFn    func(ctx context.Context, arg db.ListUserTweetsParams) ([]db.ListUserTweetsRow, error)

	// Notification
	ListNotificationsFn          func(ctx context.Context, arg db.ListNotificationsParams) ([]db.Notification, error)
	GetUnreadNotificationCountFn func(ctx context.Context, recipientID int64) (int64, error)
	MarkAllNotificationsReadFn   func(ctx context.Context, recipientID int64) error
	CreateNotificationFn         func(ctx context.Context, arg db.CreateNotificationParams) (db.Notification, error)

	// Messages
	ListUserConversationsFn          func(ctx context.Context, arg db.ListUserConversationsParams) ([]db.ListUserConversationsRow, error)
	ListConversationMessagesFn       func(ctx context.Context, arg db.ListConversationMessagesParams) ([]db.DirectMessage, error)
	CreateConversationFn             func(ctx context.Context) (db.Conversation, error)
	AddConversationParticipantFn     func(ctx context.Context, arg db.AddConversationParticipantParams) error
	CreateDirectMessageFn            func(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error)
	FindDirectConversationFn         func(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error)
	IsConversationParticipantFn      func(ctx context.Context, arg db.IsConversationParticipantParams) (bool, error)
	ListConversationParticipantIDsFn func(ctx context.Context, conversationID int64) ([]int64, error)
	TouchConversationFn              func(ctx context.Context, id int64) error

	// Search / Discovery
	SearchUsersFn                  func(ctx context.Context, arg db.SearchUsersParams) ([]db.SearchUsersRow, error)
	SearchTweetsFullTextFn         func(ctx context.Context, arg db.SearchTweetsFullTextParams) ([]db.SearchTweetsFullTextRow, error)
	SearchTweetsByHashtagFn        func(ctx context.Context, arg db.SearchTweetsByHashtagParams) ([]db.SearchTweetsByHashtagRow, error)
	ListSuggestedUsersFn           func(ctx context.Context, arg db.ListSuggestedUsersParams) ([]db.ListSuggestedUsersRow, error)
	ListTopUsersFn                 func(ctx context.Context, arg db.ListTopUsersParams) ([]db.User, error)
	ListRelatedTweetsByEmbeddingFn func(ctx context.Context, arg db.ListRelatedTweetsByEmbeddingParams) ([]db.ListRelatedTweetsByEmbeddingRow, error)

	// Batch
	GetUsersByIDsFn  func(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error)
	GetTweetsByIDsFn func(ctx context.Context, arg db.GetTweetsByIDsParams) ([]db.GetTweetsByIDsRow, error)
}

// Transaction management
func (m *MockStore) ExecTx(ctx context.Context, fn func(db.Querier) error) error {
	return fn(m)
}
func (m *MockStore) ExecTxAfterCommit(ctx context.Context, fn func(db.Querier) error, afterCommit func()) error {
	if err := fn(m); err != nil {
		return err
	}
	if afterCommit != nil {
		afterCommit()
	}
	return nil
}
func (m *MockStore) Ping(ctx context.Context) error {
	return nil
}

var _ db.Store = (*MockStore)(nil)

// Implement db.Querier methods by calling hooks with nil checks

func (m *MockStore) AddConversationParticipant(ctx context.Context, arg db.AddConversationParticipantParams) error {
	if m.AddConversationParticipantFn == nil {
		return nil
	}
	return m.AddConversationParticipantFn(ctx, arg)
}
func (m *MockStore) CreateConversation(ctx context.Context) (db.Conversation, error) {
	if m.CreateConversationFn == nil {
		return db.Conversation{}, nil
	}
	return m.CreateConversationFn(ctx)
}
func (m *MockStore) CreateDirectMessage(ctx context.Context, arg db.CreateDirectMessageParams) (db.DirectMessage, error) {
	if m.CreateDirectMessageFn == nil {
		return db.DirectMessage{}, nil
	}
	return m.CreateDirectMessageFn(ctx, arg)
}
func (m *MockStore) CreateNotification(ctx context.Context, arg db.CreateNotificationParams) (db.Notification, error) {
	if m.CreateNotificationFn == nil {
		return db.Notification{}, nil
	}
	return m.CreateNotificationFn(ctx, arg)
}
func (m *MockStore) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	if m.CreateRefreshTokenFn == nil {
		return db.RefreshToken{}, nil
	}
	return m.CreateRefreshTokenFn(ctx, arg)
}
func (m *MockStore) CreateRetweet(ctx context.Context, arg db.CreateRetweetParams) (db.CreateRetweetRow, error) {
	if m.CreateRetweetFn == nil {
		return db.CreateRetweetRow{}, nil
	}
	return m.CreateRetweetFn(ctx, arg)
}
func (m *MockStore) CreateTweet(ctx context.Context, arg db.CreateTweetParams) (db.Tweet, error) {
	if m.CreateTweetFn == nil {
		return db.Tweet{}, nil
	}
	return m.CreateTweetFn(ctx, arg)
}
func (m *MockStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if m.CreateUserFn == nil {
		return db.User{}, nil
	}
	return m.CreateUserFn(ctx, arg)
}
func (m *MockStore) DecrementHashtagUsageBy(ctx context.Context, arg db.DecrementHashtagUsageByParams) error {
	if m.DecrementHashtagUsageByFn == nil {
		return nil
	}
	return m.DecrementHashtagUsageByFn(ctx, arg)
}
func (m *MockStore) DecrementParentReplyCount(ctx context.Context, id int64) error {
	if m.DecrementParentReplyCountFn == nil {
		return nil
	}
	return m.DecrementParentReplyCountFn(ctx, id)
}
func (m *MockStore) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	if m.DeleteRefreshTokenFn == nil {
		return nil
	}
	return m.DeleteRefreshTokenFn(ctx, tokenHash)
}
func (m *MockStore) DeleteRefreshTokensByUser(ctx context.Context, userID int64) error {
	if m.DeleteRefreshTokensByUserFn == nil {
		return nil
	}
	return m.DeleteRefreshTokensByUserFn(ctx, userID)
}
func (m *MockStore) DeleteRetweetByUser(ctx context.Context, arg db.DeleteRetweetByUserParams) (db.DeleteRetweetByUserRow, error) {
	if m.DeleteRetweetByUserFn == nil {
		return db.DeleteRetweetByUserRow{}, nil
	}
	return m.DeleteRetweetByUserFn(ctx, arg)
}
func (m *MockStore) DeleteTweetByOwner(ctx context.Context, arg db.DeleteTweetByOwnerParams) (db.Tweet, error) {
	if m.DeleteTweetByOwnerFn == nil {
		return db.Tweet{}, nil
	}
	return m.DeleteTweetByOwnerFn(ctx, arg)
}
func (m *MockStore) DeleteUnusedHashtag(ctx context.Context, id int64) error {
	if m.DeleteUnusedHashtagFn == nil {
		return nil
	}
	return m.DeleteUnusedHashtagFn(ctx, id)
}
func (m *MockStore) FindDirectConversation(ctx context.Context, arg db.FindDirectConversationParams) (db.Conversation, error) {
	if m.FindDirectConversationFn == nil {
		return db.Conversation{}, nil
	}
	return m.FindDirectConversationFn(ctx, arg)
}
func (m *MockStore) FollowUser(ctx context.Context, arg db.FollowUserParams) (bool, error) {
	if m.FollowUserFn == nil {
		return false, nil
	}
	return m.FollowUserFn(ctx, arg)
}
func (m *MockStore) GetFollowedUserIDs(ctx context.Context, arg db.GetFollowedUserIDsParams) ([]int64, error) {
	if m.GetFollowedUserIDsFn == nil {
		return []int64{}, nil
	}
	return m.GetFollowedUserIDsFn(ctx, arg)
}
func (m *MockStore) GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	if m.GetRefreshTokenFn == nil {
		return db.RefreshToken{}, nil
	}
	return m.GetRefreshTokenFn(ctx, tokenHash)
}
func (m *MockStore) GetTopHashtagsAllTime(ctx context.Context, limit int32) ([]db.Hashtag, error) {
	if m.GetTopHashtagsAllTimeFn == nil {
		return []db.Hashtag{}, nil
	}
	return m.GetTopHashtagsAllTimeFn(ctx, limit)
}
func (m *MockStore) GetTrendingHashtagsLast24h(ctx context.Context, limit int32) ([]db.Hashtag, error) {
	if m.GetTrendingHashtagsLast24hFn == nil {
		return []db.Hashtag{}, nil
	}
	return m.GetTrendingHashtagsLast24hFn(ctx, limit)
}
func (m *MockStore) GetTweet(ctx context.Context, arg db.GetTweetParams) (db.GetTweetRow, error) {
	if m.GetTweetFn == nil {
		return db.GetTweetRow{}, nil
	}
	return m.GetTweetFn(ctx, arg)
}
func (m *MockStore) GetTweetsByIDs(ctx context.Context, arg db.GetTweetsByIDsParams) ([]db.GetTweetsByIDsRow, error) {
	if m.GetTweetsByIDsFn == nil {
		return []db.GetTweetsByIDsRow{}, nil
	}
	return m.GetTweetsByIDsFn(ctx, arg)
}
func (m *MockStore) GetUnreadNotificationCount(ctx context.Context, recipientID int64) (int64, error) {
	if m.GetUnreadNotificationCountFn == nil {
		return 0, nil
	}
	return m.GetUnreadNotificationCountFn(ctx, recipientID)
}
func (m *MockStore) GetUser(ctx context.Context, arg db.GetUserParams) (db.GetUserRow, error) {
	if m.GetUserFn == nil {
		return db.GetUserRow{}, nil
	}
	return m.GetUserFn(ctx, arg)
}
func (m *MockStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.GetUserByEmailFn == nil {
		return db.User{}, nil
	}
	return m.GetUserByEmailFn(ctx, email)
}
func (m *MockStore) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	if m.GetUserByUsernameFn == nil {
		return db.User{}, nil
	}
	return m.GetUserByUsernameFn(ctx, username)
}
func (m *MockStore) GetUserRetweet(ctx context.Context, arg db.GetUserRetweetParams) (db.Tweet, error) {
	if m.GetUserRetweetFn == nil {
		return db.Tweet{}, nil
	}
	return m.GetUserRetweetFn(ctx, arg)
}
func (m *MockStore) GetUsersByIDs(ctx context.Context, arg db.GetUsersByIDsParams) ([]db.GetUsersByIDsRow, error) {
	if m.GetUsersByIDsFn == nil {
		return []db.GetUsersByIDsRow{}, nil
	}
	return m.GetUsersByIDsFn(ctx, arg)
}
func (m *MockStore) IncrementParentReplyCount(ctx context.Context, id int64) error {
	if m.IncrementParentReplyCountFn == nil {
		return nil
	}
	return m.IncrementParentReplyCountFn(ctx, id)
}
func (m *MockStore) IsConversationParticipant(ctx context.Context, arg db.IsConversationParticipantParams) (bool, error) {
	if m.IsConversationParticipantFn == nil {
		return false, nil
	}
	return m.IsConversationParticipantFn(ctx, arg)
}
func (m *MockStore) IsFollowing(ctx context.Context, arg db.IsFollowingParams) (bool, error) {
	if m.IsFollowingFn == nil {
		return false, nil
	}
	return m.IsFollowingFn(ctx, arg)
}
func (m *MockStore) IsTweetLiked(ctx context.Context, arg db.IsTweetLikedParams) (bool, error) {
	if m.IsTweetLikedFn == nil {
		return false, nil
	}
	return m.IsTweetLikedFn(ctx, arg)
}
func (m *MockStore) LikeTweet(ctx context.Context, arg db.LikeTweetParams) (bool, error) {
	if m.LikeTweetFn == nil {
		return false, nil
	}
	return m.LikeTweetFn(ctx, arg)
}
func (m *MockStore) LinkTweetHashtag(ctx context.Context, arg db.LinkTweetHashtagParams) error {
	if m.LinkTweetHashtagFn == nil {
		return nil
	}
	return m.LinkTweetHashtagFn(ctx, arg)
}
func (m *MockStore) ListConversationMessages(ctx context.Context, arg db.ListConversationMessagesParams) ([]db.DirectMessage, error) {
	if m.ListConversationMessagesFn == nil {
		return []db.DirectMessage{}, nil
	}
	return m.ListConversationMessagesFn(ctx, arg)
}
func (m *MockStore) ListConversationParticipantIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	if m.ListConversationParticipantIDsFn == nil {
		return []int64{}, nil
	}
	return m.ListConversationParticipantIDsFn(ctx, conversationID)
}
func (m *MockStore) ListFollowersUsers(ctx context.Context, arg db.ListFollowersUsersParams) ([]db.ListFollowersUsersRow, error) {
	if m.ListFollowersUsersFn == nil {
		return []db.ListFollowersUsersRow{}, nil
	}
	return m.ListFollowersUsersFn(ctx, arg)
}
func (m *MockStore) ListFollowingFeed(ctx context.Context, arg db.ListFollowingFeedParams) ([]db.ListFollowingFeedRow, error) {
	if m.ListFollowingFeedFn == nil {
		return []db.ListFollowingFeedRow{}, nil
	}
	return m.ListFollowingFeedFn(ctx, arg)
}
func (m *MockStore) ListFollowingUsers(ctx context.Context, arg db.ListFollowingUsersParams) ([]db.ListFollowingUsersRow, error) {
	if m.ListFollowingUsersFn == nil {
		return []db.ListFollowingUsersRow{}, nil
	}
	return m.ListFollowingUsersFn(ctx, arg)
}
func (m *MockStore) ListForYouFeed(ctx context.Context, arg db.ListForYouFeedParams) ([]db.ListForYouFeedRow, error) {
	if m.ListForYouFeedFn == nil {
		return []db.ListForYouFeedRow{}, nil
	}
	return m.ListForYouFeedFn(ctx, arg)
}
func (m *MockStore) ListHashtagUsageToDecrementForDeleteRoot(ctx context.Context, id int64) ([]db.ListHashtagUsageToDecrementForDeleteRootRow, error) {
	if m.ListHashtagUsageToDecrementForDeleteRootFn == nil {
		return []db.ListHashtagUsageToDecrementForDeleteRootRow{}, nil
	}
	return m.ListHashtagUsageToDecrementForDeleteRootFn(ctx, id)
}
func (m *MockStore) ListMediaURLsInThread(ctx context.Context, id int64) ([]*string, error) {
	if m.ListMediaURLsInThreadFn == nil {
		return []*string{}, nil
	}
	return m.ListMediaURLsInThreadFn(ctx, id)
}
func (m *MockStore) ListNotifications(ctx context.Context, arg db.ListNotificationsParams) ([]db.Notification, error) {
	if m.ListNotificationsFn == nil {
		return []db.Notification{}, nil
	}
	return m.ListNotificationsFn(ctx, arg)
}
func (m *MockStore) ListRelatedTweetsByEmbedding(ctx context.Context, arg db.ListRelatedTweetsByEmbeddingParams) ([]db.ListRelatedTweetsByEmbeddingRow, error) {
	if m.ListRelatedTweetsByEmbeddingFn == nil {
		return []db.ListRelatedTweetsByEmbeddingRow{}, nil
	}
	return m.ListRelatedTweetsByEmbeddingFn(ctx, arg)
}
func (m *MockStore) ListSuggestedUsers(ctx context.Context, arg db.ListSuggestedUsersParams) ([]db.ListSuggestedUsersRow, error) {
	if m.ListSuggestedUsersFn == nil {
		return []db.ListSuggestedUsersRow{}, nil
	}
	return m.ListSuggestedUsersFn(ctx, arg)
}
func (m *MockStore) ListTopUsers(ctx context.Context, arg db.ListTopUsersParams) ([]db.User, error) {
	if m.ListTopUsersFn == nil {
		return []db.User{}, nil
	}
	return m.ListTopUsersFn(ctx, arg)
}
func (m *MockStore) ListTweetReplies(ctx context.Context, arg db.ListTweetRepliesParams) ([]db.ListTweetRepliesRow, error) {
	if m.ListTweetRepliesFn == nil {
		return []db.ListTweetRepliesRow{}, nil
	}
	return m.ListTweetRepliesFn(ctx, arg)
}
func (m *MockStore) ListUserConversations(ctx context.Context, arg db.ListUserConversationsParams) ([]db.ListUserConversationsRow, error) {
	if m.ListUserConversationsFn == nil {
		return []db.ListUserConversationsRow{}, nil
	}
	return m.ListUserConversationsFn(ctx, arg)
}
func (m *MockStore) ListUserTweets(ctx context.Context, arg db.ListUserTweetsParams) ([]db.ListUserTweetsRow, error) {
	if m.ListUserTweetsFn == nil {
		return []db.ListUserTweetsRow{}, nil
	}
	return m.ListUserTweetsFn(ctx, arg)
}
func (m *MockStore) MarkAllNotificationsRead(ctx context.Context, recipientID int64) error {
	if m.MarkAllNotificationsReadFn == nil {
		return nil
	}
	return m.MarkAllNotificationsReadFn(ctx, recipientID)
}
func (m *MockStore) SearchHashtagsByPrefix(ctx context.Context, arg db.SearchHashtagsByPrefixParams) ([]db.Hashtag, error) {
	if m.SearchHashtagsByPrefixFn == nil {
		return []db.Hashtag{}, nil
	}
	return m.SearchHashtagsByPrefixFn(ctx, arg)
}
func (m *MockStore) SearchTweetsByHashtag(ctx context.Context, arg db.SearchTweetsByHashtagParams) ([]db.SearchTweetsByHashtagRow, error) {
	if m.SearchTweetsByHashtagFn == nil {
		return []db.SearchTweetsByHashtagRow{}, nil
	}
	return m.SearchTweetsByHashtagFn(ctx, arg)
}
func (m *MockStore) SearchTweetsFullText(ctx context.Context, arg db.SearchTweetsFullTextParams) ([]db.SearchTweetsFullTextRow, error) {
	if m.SearchTweetsFullTextFn == nil {
		return []db.SearchTweetsFullTextRow{}, nil
	}
	return m.SearchTweetsFullTextFn(ctx, arg)
}
func (m *MockStore) SearchUsers(ctx context.Context, arg db.SearchUsersParams) ([]db.SearchUsersRow, error) {
	if m.SearchUsersFn == nil {
		return []db.SearchUsersRow{}, nil
	}
	return m.SearchUsersFn(ctx, arg)
}
func (m *MockStore) TouchConversation(ctx context.Context, id int64) error {
	if m.TouchConversationFn == nil {
		return nil
	}
	return m.TouchConversationFn(ctx, id)
}
func (m *MockStore) UnfollowUser(ctx context.Context, arg db.UnfollowUserParams) (bool, error) {
	if m.UnfollowUserFn == nil {
		return false, nil
	}
	return m.UnfollowUserFn(ctx, arg)
}
func (m *MockStore) UnlikeTweet(ctx context.Context, arg db.UnlikeTweetParams) (bool, error) {
	if m.UnlikeTweetFn == nil {
		return false, nil
	}
	return m.UnlikeTweetFn(ctx, arg)
}
func (m *MockStore) UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error) {
	if m.UpdateUserProfileFn == nil {
		return db.User{}, nil
	}
	return m.UpdateUserProfileFn(ctx, arg)
}
func (m *MockStore) UpsertHashtag(ctx context.Context, text string) (db.Hashtag, error) {
	if m.UpsertHashtagFn == nil {
		return db.Hashtag{}, nil
	}
	return m.UpsertHashtagFn(ctx, text)
}

// Ptr returns a pointer to v (handy for nullable sqlc fields).
func Ptr[T any](v T) *T { return &v }

// Ensure interface Satisfaction
var _ db.Store = (*MockStore)(nil)
