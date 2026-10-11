package api

import (
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
)

// ErrorResponse is a local alias for Swaggo readability.
type ErrorResponse = apperr.ErrorResponse

// CurrentUserResponse is the authenticated user's own profile, including private fields.
type CurrentUserResponse struct {
	UserResponse
	Email string `json:"email"`
	// Onboarded is false until the user has chosen their own username.
	Onboarded bool `json:"onboarded"`
}

func newCurrentUserResponse(user domain.User) CurrentUserResponse {
	return CurrentUserResponse{UserResponse: newUserResponse(user), Email: user.Email, Onboarded: user.Onboarded}
}

type UserResponse struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	DisplayName    *string   `json:"displayName"`
	Bio            *string   `json:"bio"`
	Location       *string   `json:"location"`
	Website        *string   `json:"website"`
	AvatarURL      *string   `json:"avatarUrl"`
	BannerURL      *string   `json:"bannerUrl"`
	IsFollowing    bool      `json:"isFollowing"`
	FollowersCount int32     `json:"followersCount"`
	FollowingCount int32     `json:"followingCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func newUserResponse(user domain.User) UserResponse {
	return UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		Location:       user.Location,
		Website:        user.Website,
		AvatarURL:      user.AvatarURL,
		BannerURL:      user.BannerURL,
		IsFollowing:    user.IsFollowing,
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		CreatedAt:      user.CreatedAt,
	}
}

type TweetResponse struct {
	ID              int64        `json:"id"`
	Content         *string      `json:"content"`
	MediaType       *string      `json:"mediaType"`
	MediaURL        *string      `json:"mediaUrl"`
	User            UserResponse `json:"user"`
	ReplyCount      int32        `json:"replyCount"`
	LikeCount       int32        `json:"likeCount"`
	RetweetCount    int32        `json:"retweetCount"`
	IsLiked         bool         `json:"isLiked"`
	IsRetweeted     bool         `json:"isRetweeted"`
	ReplyToTweetID  *int64       `json:"replyToTweetId"`
	ReplyToUsername *string      `json:"replyToUsername"`
	CreatedAt       time.Time    `json:"createdAt"`
}

// FeedItemResponse is one entry of a timeline (feeds and profile tabs). kind is
// "retweet" when the tweet is there because someone retweeted it; retweet is then
// set (and null otherwise). The tweet is always the original, in the same shape as
// GET /tweets/{id}.
type FeedItemResponse struct {
	Kind    string           `json:"kind" enums:"tweet,retweet"`
	Tweet   TweetResponse    `json:"tweet"`
	Retweet *RetweetResponse `json:"retweet"`
}

type RetweetResponse struct {
	ID        int64        `json:"id"`
	Retweeter UserResponse `json:"retweeter"`
	CreatedAt time.Time    `json:"createdAt"`
}

func newFeedItemResponse(item domain.FeedItem) FeedItemResponse {
	resp := FeedItemResponse{Kind: "tweet", Tweet: newTweetResponse(item.Tweet)}
	if item.Retweet != nil {
		resp.Kind = "retweet"
		resp.Retweet = &RetweetResponse{
			ID:        item.Retweet.ID,
			Retweeter: newUserResponse(item.Retweet.By),
			CreatedAt: item.Retweet.CreatedAt,
		}
	}
	return resp
}

func newTweetResponse(tweet domain.Tweet) TweetResponse {
	return TweetResponse{
		ID:              tweet.ID,
		Content:         tweet.Content,
		MediaType:       tweet.MediaType,
		MediaURL:        tweet.MediaURL,
		User:            newUserResponse(tweet.Author),
		ReplyCount:      tweet.ReplyCount,
		LikeCount:       tweet.LikeCount,
		RetweetCount:    tweet.RetweetCount,
		IsLiked:         tweet.IsLiked,
		IsRetweeted:     tweet.IsRetweeted,
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

func newHashtagResponse(tag domain.Hashtag) HashtagResponse {
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
	TweetMediaURL         *string      `json:"tweetMediaUrl"`
	OriginalTweetID       *int64       `json:"originalTweetId,omitempty"`
	OriginalTweetContent  *string      `json:"originalTweetContent,omitempty"`
	OriginalTweetMediaURL *string      `json:"originalTweetMediaUrl,omitempty"`
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

func newNotificationResponse(item domain.Notification) NotificationResponse {
	return NotificationResponse{
		ID:                    item.ID,
		Actor:                 newUserResponse(item.Actor),
		TweetID:               item.TweetID,
		TweetContent:          item.TweetContent,
		TweetMediaURL:         item.TweetMediaURL,
		OriginalTweetID:       item.OriginalTweetID,
		OriginalTweetContent:  item.OriginalTweetContent,
		OriginalTweetMediaURL: item.OriginalTweetMediaURL,
		Type:                  string(item.Type),
		IsRead:                item.IsRead,
		CreatedAt:             item.CreatedAt,
	}
}

func newMessageResponse(item domain.Message) MessageResponse {
	return MessageResponse{
		ID:             item.ID,
		ConversationID: item.ConversationID,
		Sender:         newUserResponse(item.Sender),
		Content:        item.Content,
		CreatedAt:      item.CreatedAt,
	}
}

func newConversationResponse(item domain.Conversation) ConversationResponse {
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

func newHashtagResponseList(hashtags []domain.Hashtag) []HashtagResponse {
	response := make([]HashtagResponse, 0, len(hashtags))
	for _, h := range hashtags {
		response = append(response, newHashtagResponse(h))
	}
	return response
}
