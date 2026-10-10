package server

import (
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/usecase"
)

// ErrorResponse is a local alias for Swaggo readability.
type ErrorResponse = apperr.ErrorResponse

type AuthResponse struct {
	AccessToken  string              `json:"accessToken"`
	RefreshToken string              `json:"refreshToken"`
	User         CurrentUserResponse `json:"user"`
}

func newAuthResponse(accessToken, refreshToken string, user usecase.UserItem) AuthResponse {
	return AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         newCurrentUserResponse(user),
	}
}

// CurrentUserResponse is the authenticated user's own profile, including private fields.
type CurrentUserResponse struct {
	UserResponse
	Email string `json:"email"`
}

func newCurrentUserResponse(user usecase.UserItem) CurrentUserResponse {
	return CurrentUserResponse{UserResponse: newUserResponse(user), Email: user.Email}
}

type UserResponse struct {
	ID             int64   `json:"id"`
	Username       string  `json:"username"`
	DisplayName    *string `json:"displayName"`
	Bio            *string `json:"bio"`
	AvatarUrl      *string `json:"avatarUrl"`
	BannerUrl      *string `json:"bannerUrl"`
	IsFollowing    bool    `json:"isFollowing"`
	FollowersCount int32   `json:"followersCount"`
	FollowingCount int32   `json:"followingCount"`
}

func newUserResponse(user usecase.UserItem) UserResponse {
	return UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		AvatarUrl:      user.AvatarUrl,
		BannerUrl:      user.BannerUrl,
		IsFollowing:    user.IsFollowing,
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
	}
}

type TweetResponse struct {
	ID              int64          `json:"id"`
	Content         *string        `json:"content"`
	MediaType       *string        `json:"mediaType"`
	MediaUrl        *string        `json:"mediaUrl"`
	User            UserResponse   `json:"user"`
	ReplyCount      int32          `json:"replyCount"`
	LikeCount       int32          `json:"likeCount"`
	RetweetCount    int32          `json:"retweetCount"`
	IsLiked         bool           `json:"isLiked"`
	IsRetweeted     bool           `json:"isRetweeted"`
	RetweetedTweet  *TweetResponse `json:"retweetedTweet,omitempty"`
	ReplyToTweetID  *int64         `json:"replyToTweetId"`
	ReplyToUsername *string        `json:"replyToUsername"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func newTweetResponse(tweet usecase.TweetItem) TweetResponse {
	var original *TweetResponse
	if tweet.OriginalTweet != nil {
		r := newTweetResponse(*tweet.OriginalTweet)
		original = &r
	}

	return TweetResponse{
		ID:              tweet.ID,
		Content:         tweet.Content,
		MediaType:       tweet.MediaType,
		MediaUrl:        tweet.MediaUrl,
		User:            newUserResponse(tweet.Author),
		ReplyCount:      tweet.ReplyCount,
		LikeCount:       tweet.LikeCount,
		RetweetCount:    tweet.RetweetCount,
		IsLiked:         tweet.IsLiked,
		IsRetweeted:     tweet.IsRetweeted,
		RetweetedTweet:  original,
		ReplyToTweetID:  tweet.ParentID,
		ReplyToUsername: tweet.ParentUsername,
		CreatedAt:       tweet.CreatedAt,
	}
}

type HashtagResponse struct {
	ID         int64     `json:"id"`
	Text       string    `json:"text"`
	UsageCount int32     `json:"usageCount"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

func newHashtagResponse(tag db.Hashtag) HashtagResponse {
	return HashtagResponse{
		ID:         tag.ID,
		Text:       tag.Text,
		UsageCount: tag.UsageCount,
		LastUsedAt: tag.LastUsedAt,
		CreatedAt:  tag.CreatedAt,
	}
}

type NotificationResponse struct {
	ID                    int64        `json:"id"`
	Actor                 UserResponse `json:"actor"`
	TweetID               *int64       `json:"tweetId"`
	TweetContent          *string      `json:"tweetContent"`
	TweetMediaUrl         *string      `json:"tweetMediaUrl"`
	OriginalTweetID       *int64       `json:"originalTweetId,omitempty"`
	OriginalTweetContent  *string      `json:"originalTweetContent,omitempty"`
	OriginalTweetMediaUrl *string      `json:"originalTweetMediaUrl,omitempty"`
	Type                  string       `json:"type"`
	IsRead                bool         `json:"isRead"`
	CreatedAt             time.Time    `json:"createdAt"`
}

type MessageResponse struct {
	ID             int64        `json:"id"`
	ConversationID int64        `json:"conversationId"`
	Sender         UserResponse `json:"sender"`
	Content        string       `json:"content"`
	CreatedAt      time.Time    `json:"createdAt"`
}



type ConversationResponse struct {
	ID          int64           `json:"id"`
	Peer        UserResponse    `json:"peer"`
	LastMessage MessageResponse `json:"lastMessage"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

func newNotificationResponse(item usecase.NotificationItem) NotificationResponse {
	return NotificationResponse{
		ID:                    item.ID,
		Actor:                 newUserResponse(item.Actor),
		TweetID:               item.TweetID,
		TweetContent:          item.TweetContent,
		TweetMediaUrl:         item.TweetMediaUrl,
		OriginalTweetID:       item.OriginalTweetID,
		OriginalTweetContent:  item.OriginalTweetContent,
		OriginalTweetMediaUrl: item.OriginalTweetMediaUrl,
		Type:                  item.Type,
		IsRead:                item.IsRead,
		CreatedAt:             item.CreatedAt,
	}
}

func newMessageResponse(item usecase.MessageItem) MessageResponse {
	return MessageResponse{
		ID:             item.ID,
		ConversationID: item.ConversationID,
		Sender:         newUserResponse(item.Sender),
		Content:        item.Content,
		CreatedAt:      item.CreatedAt,
	}
}

func newConversationResponse(item usecase.ConversationItem) ConversationResponse {
	return ConversationResponse{
		ID:          item.ID,
		Peer:        newUserResponse(item.Peer),
		LastMessage: newMessageResponse(item.LastMessage),
		UpdatedAt:   item.UpdatedAt,
	}
}



type SuccessResponse struct {
	Success bool `json:"success"`
}

func successResponse() SuccessResponse {
	return SuccessResponse{Success: true}
}

func newUserResponseList(users []usecase.UserItem) []UserResponse {
	response := make([]UserResponse, 0, len(users))
	for _, user := range users {
		response = append(response, newUserResponse(user))
	}
	return response
}

func newTweetResponseList(tweets []usecase.TweetItem) []TweetResponse {
	response := make([]TweetResponse, 0, len(tweets))
	for _, t := range tweets {
		response = append(response, newTweetResponse(t))
	}
	return response
}

func newHashtagResponseList(hashtags []db.Hashtag) []HashtagResponse {
	response := make([]HashtagResponse, 0, len(hashtags))
	for _, h := range hashtags {
		response = append(response, newHashtagResponse(h))
	}
	return response
}

func newNotificationResponseList(items []usecase.NotificationItem) []NotificationResponse {
	response := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, newNotificationResponse(item))
	}
	return response
}

func newMessageResponseList(items []usecase.MessageItem) []MessageResponse {
	response := make([]MessageResponse, 0, len(items))
	for _, item := range items {
		response = append(response, newMessageResponse(item))
	}
	return response
}

func newConversationResponseList(items []usecase.ConversationItem) []ConversationResponse {
	response := make([]ConversationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, newConversationResponse(item))
	}
	return response
}

