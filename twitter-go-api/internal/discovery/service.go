// Package discovery suggests what to read and who to follow.
package discovery

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
)

type Service struct {
	store db.Store
}

func New(store db.Store) *Service {
	return &Service{store: store}
}

// TrendingHashtags ranks hashtags by use in the last 24h, falling back to all-time usage when it's quiet.
func (s *Service) TrendingHashtags(ctx context.Context, limit int32) ([]domain.Hashtag, error) {
	rows, err := s.store.GetTrendingHashtagsLast24h(ctx, limit)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if rows, err = s.store.GetTopHashtagsAllTime(ctx, limit); err != nil {
			return nil, err
		}
	}
	return domain.NewHashtags(rows), nil
}

// SuggestedUsers favours accounts the viewer doesn't follow yet; anonymous viewers get the most-followed users.
func (s *Service) SuggestedUsers(ctx context.Context, page, size int32, viewerID *int64) ([]domain.User, error) {
	if viewerID == nil {
		rows, err := s.store.ListTopUsers(ctx, db.ListTopUsersParams{Limit: size, Offset: page * size})
		if err != nil {
			return nil, err
		}
		users := make([]domain.User, 0, len(rows))
		for _, u := range rows {
			users = append(users, domain.NewUser(u, false))
		}
		return users, nil
	}

	rows, err := s.store.ListSuggestedUsers(ctx, db.ListSuggestedUsersParams{FollowerID: *viewerID, Limit: size, Offset: page * size, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, domain.NewUser(r.User, r.IsFollowing))
	}
	return users, nil
}
