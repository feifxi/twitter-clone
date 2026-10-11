// Package discovery suggests what to read and who to follow.
package discovery

import (
	"context"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
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

// SuggestedUsers lists the most-followed accounts the viewer doesn't follow yet; it pages by offset.
func (s *Service) SuggestedUsers(ctx context.Context, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error) {
	rows, err := s.store.ListSuggestedUsers(ctx, db.ListSuggestedUsersParams{
		ViewerID:   viewerID,
		PageLimit:  req.FetchLimit(),
		PageOffset: req.Offset(),
	})
	if err != nil {
		return pagination.Page[domain.User]{}, err
	}

	rows, more := pagination.Trim(rows, req.Limit)
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, domain.NewUser(r.User, r.IsFollowing))
	}
	page := pagination.Page[domain.User]{Items: users}
	if more {
		next := req.NextOffset()
		page.Next = &next
	}
	return page, nil
}
