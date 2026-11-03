// Package search finds users, tweets (full-text or by hashtag) and hashtags.
package search

import (
	"context"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
)

type Service struct {
	store db.Store
}

func New(store db.Store) *Service {
	return &Service{store: store}
}

// Users ranks matches by popularity; it pages by offset.
func (s *Service) Users(ctx context.Context, query string, req pagination.Request, viewerID *int64) (pagination.Page[domain.User], error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return pagination.Page[domain.User]{Items: []domain.User{}}, nil
	}

	rows, err := s.store.SearchUsers(ctx, db.SearchUsersParams{
		Query:      &q,
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
	return pagination.Page[domain.User]{Items: users, Next: nextOffset(req, more)}, nil
}

// Tweets runs a hashtag search (newest first) when the query starts with '#',
// otherwise a relevance-ranked full-text search (paged by offset).
func (s *Service) Tweets(ctx context.Context, query string, req pagination.Request, viewerID *int64) (pagination.Page[domain.Tweet], error) {
	empty := pagination.Page[domain.Tweet]{Items: []domain.Tweet{}}
	q := strings.TrimSpace(query)

	if tag, ok := strings.CutPrefix(q, "#"); ok {
		tag = strings.TrimLeft(tag, "#")
		if tag == "" {
			return empty, nil
		}
		rows, err := s.store.SearchTweetsByHashtag(ctx, db.SearchTweetsByHashtagParams{
			Tag:        tag,
			ViewerID:   viewerID,
			CursorTime: req.AfterTime(),
			CursorID:   req.AfterID(),
			PageLimit:  req.FetchLimit(),
		})
		if err != nil {
			return pagination.Page[domain.Tweet]{}, err
		}
		return tweet.Page(ctx, s.store, tweet.Rows(rows, func(r db.SearchTweetsByHashtagRow) tweet.Row { return tweet.Row(r) }), req.Limit, viewerID, tweet.CreatedCursor)
	}

	tsQuery := buildTSQuery(q)
	if tsQuery == "" {
		return empty, nil
	}
	rows, err := s.store.SearchTweetsFullText(ctx, db.SearchTweetsFullTextParams{
		Query:      tsQuery,
		ViewerID:   viewerID,
		PageLimit:  req.FetchLimit(),
		PageOffset: req.Offset(),
	})
	if err != nil {
		return pagination.Page[domain.Tweet]{}, err
	}

	rows, more := pagination.Trim(rows, req.Limit)
	items, err := tweet.Hydrate(ctx, s.store, tweet.Rows(rows, func(r db.SearchTweetsFullTextRow) tweet.Row { return tweet.Row(r) }), viewerID)
	if err != nil {
		return pagination.Page[domain.Tweet]{}, err
	}
	return pagination.Page[domain.Tweet]{Items: items, Next: nextOffset(req, more)}, nil
}

func nextOffset(req pagination.Request, more bool) *pagination.Cursor {
	if !more {
		return nil
	}
	c := req.NextOffset()
	return &c
}

func (s *Service) Hashtags(ctx context.Context, query string, limit int32) ([]domain.Hashtag, error) {
	prefix := strings.TrimPrefix(strings.TrimSpace(query), "#")
	if prefix == "" {
		return []domain.Hashtag{}, nil
	}
	rows, err := s.store.SearchHashtagsByPrefix(ctx, db.SearchHashtagsByPrefixParams{Prefix: prefix, ResultLimit: limit})
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
