// Package search finds users, tweets (full-text or by hashtag) and hashtags.
package search

import (
	"context"
	"strings"

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

func (s *Service) Users(ctx context.Context, query string, page, size int32, viewerID *int64) ([]domain.User, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []domain.User{}, nil
	}

	rows, err := s.store.SearchUsers(ctx, db.SearchUsersParams{Column1: &q, Limit: size, Offset: page * size, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		users = append(users, domain.NewUser(r.User, r.IsFollowing))
	}
	return users, nil
}

// Tweets runs a hashtag search when the query starts with '#', otherwise a full-text search.
func (s *Service) Tweets(ctx context.Context, query string, page, size int32, viewerID *int64) ([]domain.Tweet, error) {
	q := strings.TrimSpace(query)

	if tag, ok := strings.CutPrefix(q, "#"); ok {
		tag = strings.ToLower(strings.TrimLeft(tag, "#"))
		if tag == "" {
			return []domain.Tweet{}, nil
		}
		rows, err := s.store.SearchTweetsByHashtag(ctx, db.SearchTweetsByHashtagParams{Lower: tag, Limit: size, Offset: page * size, ViewerID: viewerID})
		if err != nil {
			return nil, err
		}
		return tweet.Hydrate(ctx, s.store, rows, viewerID, func(r db.SearchTweetsByHashtagRow) tweet.Row { return tweet.Row(r) })
	}

	tsQuery := buildTSQuery(q)
	if tsQuery == "" {
		return []domain.Tweet{}, nil
	}
	rows, err := s.store.SearchTweetsFullText(ctx, db.SearchTweetsFullTextParams{ToTsquery: tsQuery, Limit: size, Offset: page * size, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	return tweet.Hydrate(ctx, s.store, rows, viewerID, func(r db.SearchTweetsFullTextRow) tweet.Row { return tweet.Row(r) })
}

func (s *Service) Hashtags(ctx context.Context, query string, limit int32) ([]domain.Hashtag, error) {
	prefix := strings.TrimPrefix(strings.TrimSpace(query), "#")
	if prefix == "" {
		return []domain.Hashtag{}, nil
	}
	rows, err := s.store.SearchHashtagsByPrefix(ctx, db.SearchHashtagsByPrefixParams{Column1: &prefix, Limit: limit})
	if err != nil {
		return nil, err
	}
	return domain.NewHashtags(rows), nil
}

// buildTSQuery ANDs the alphanumeric words of raw; everything else is dropped so
// user input can't inject tsquery operators.
func buildTSQuery(raw string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, raw)
	return strings.Join(strings.Fields(clean), " & ")
}
