// Package domain holds the read models shared across feature packages. It sits at
// the bottom of the import graph so features (user, tweet, notification…) can
// exchange these types without importing each other.
package domain

import (
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
)

type NotificationType string

const (
	NotificationReply   NotificationType = "REPLY"
	NotificationLike    NotificationType = "LIKE"
	NotificationRetweet NotificationType = "RETWEET"
	NotificationFollow  NotificationType = "FOLLOW"
)

const (
	MediaImage = "IMAGE"
	MediaVideo = "VIDEO"
)

const RoleUser = "USER"

type User struct {
	ID             int64
	Username       string
	Email          string
	DisplayName    *string
	Bio            *string
	Location       *string
	Website        *string
	AvatarURL      *string
	BannerURL      *string
	Role           string
	Provider       string
	FollowersCount int32
	FollowingCount int32
	// Onboarded is false until the user picks their own username (sign-up assigns a generated one).
	Onboarded   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsFollowing bool
}

type Tweet struct {
	ID             int64
	UserID         int64
	Content        *string
	MediaType      *string
	MediaURL       *string
	ParentID       *int64
	ReplyCount     int32
	RetweetCount   int32
	LikeCount      int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Author         User
	IsLiked        bool
	IsRetweeted    bool
	IsFollowing    bool
	ParentUsername *string
	// RetweetedBy is set when the tweet appears in a timeline because someone retweeted it.
	RetweetedBy *User
	RetweetedAt *time.Time
}

type Hashtag struct {
	ID         int64
	Text       string
	UsageCount int32
	LastUsedAt time.Time
	CreatedAt  time.Time
}

type Notification struct {
	ID                    int64
	RecipientID           int64
	Type                  NotificationType
	IsRead                bool
	CreatedAt             time.Time
	Actor                 User
	TweetID               *int64
	TweetContent          *string
	TweetMediaURL         *string
	OriginalTweetID       *int64
	OriginalTweetContent  *string
	OriginalTweetMediaURL *string
}

type Message struct {
	ID             int64
	ConversationID int64
	Sender         User
	Content        string
	CreatedAt      time.Time
}

type Conversation struct {
	ID          int64
	Peer        User
	LastMessage Message
	UpdatedAt   time.Time
}

func NewUser(u db.User, isFollowing bool) User {
	return User{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		Bio:            u.Bio,
		Location:       u.Location,
		Website:        u.Website,
		AvatarURL:      u.AvatarURL,
		BannerURL:      u.BannerURL,
		Role:           u.Role,
		Provider:       u.Provider,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		Onboarded:      u.OnboardedAt != nil,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
		IsFollowing:    isFollowing,
	}
}

// NewTweet maps the row only; author, viewer flags and timeline context are filled in by tweet hydration.
func NewTweet(t db.Tweet) Tweet {
	return Tweet{
		ID:           t.ID,
		UserID:       t.UserID,
		Content:      t.Content,
		MediaType:    t.MediaType,
		MediaURL:     t.MediaURL,
		ParentID:     t.ParentID,
		ReplyCount:   t.ReplyCount,
		RetweetCount: t.RetweetCount,
		LikeCount:    t.LikeCount,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func NewHashtags(rows []db.Hashtag) []Hashtag {
	out := make([]Hashtag, 0, len(rows))
	for _, h := range rows {
		out = append(out, Hashtag{
			ID:         h.ID,
			Text:       h.Text,
			UsageCount: h.UsageCount,
			LastUsedAt: h.LastUsedAt,
			CreatedAt:  h.CreatedAt,
		})
	}
	return out
}
