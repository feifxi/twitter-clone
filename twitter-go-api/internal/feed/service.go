// Package feed serves the home timelines and a user's profile timeline.
package feed

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/tweet"
)

type Service struct {
	store db.Store
}

func New(store db.Store) *Service {
	return &Service{store: store}
}

// Global is the ranked "For You" feed of top-level tweets.
func (s *Service) Global(ctx context.Context, page, size int32, viewerID *int64) ([]domain.Tweet, error) {
	rows, err := s.store.ListForYouFeed(ctx, db.ListForYouFeedParams{Limit: size, Offset: page * size, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	return tweet.Hydrate(ctx, s.store, rows, viewerID, func(r db.ListForYouFeedRow) tweet.Row { return tweet.Row(r) })
}

// Following is the chronological feed of accounts userID follows.
func (s *Service) Following(ctx context.Context, userID int64, page, size int32) ([]domain.Tweet, error) {
	rows, err := s.store.ListFollowingFeed(ctx, db.ListFollowingFeedParams{
		FollowerID: userID,
		Limit:      size,
		Offset:     page * size,
		ViewerID:   &userID,
	})
	if err != nil {
		return nil, err
	}
	return tweet.Hydrate(ctx, s.store, rows, &userID, func(r db.ListFollowingFeedRow) tweet.Row { return tweet.Row(r) })
}

// ForUser is userID's profile timeline (their tweets and retweets).
func (s *Service) ForUser(ctx context.Context, userID int64, page, size int32, viewerID *int64) ([]domain.Tweet, error) {
	if _, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID, ViewerID: viewerID}); err != nil {
		return nil, err
	}

	rows, err := s.store.ListUserTweets(ctx, db.ListUserTweetsParams{
		UserID:   userID,
		Limit:    size,
		Offset:   page * size,
		ViewerID: viewerID,
	})
	if err != nil {
		return nil, err
	}
	return tweet.Hydrate(ctx, s.store, rows, viewerID, func(r db.ListUserTweetsRow) tweet.Row { return tweet.Row(r) })
}
