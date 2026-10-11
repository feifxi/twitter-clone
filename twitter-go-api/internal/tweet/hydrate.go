package tweet

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
)

// Row is the shape shared by every tweet-listing query: the tweet, the viewer's
// flags and, for timelines, the retweet that put it there. Each sqlc row type
// with these columns converts to it directly: Row(r).
type Row struct {
	Tweet         db.Tweet
	IsLiked       bool
	IsRetweeted   bool
	IsFollowing   bool
	RetweetID     *int64
	RetweetedByID *int64
	RetweetedAt   *time.Time
}

// TimelineCursor is the sort key of a timeline row: the retweet's time for
// retweets, the tweet's own time otherwise (see ListFollowingTimeline).
func TimelineCursor(r Row) pagination.Cursor {
	if r.RetweetID != nil && r.RetweetedAt != nil {
		return pagination.Cursor{Time: *r.RetweetedAt, Kind: 1, ID: *r.RetweetID}
	}
	return pagination.Cursor{Time: r.Tweet.CreatedAt, ID: r.Tweet.ID}
}

// CreatedCursor is the sort key for lists ordered by (created_at, id).
func CreatedCursor(r Row) pagination.Cursor {
	return pagination.Cursor{Time: r.Tweet.CreatedAt, ID: r.Tweet.ID}
}

// Hydrate turns rows into domain tweets with authors and reply targets filled in.
// For timelines (which can contain retweets), use HydrateItems.
func Hydrate(ctx context.Context, store db.Querier, rows []Row, viewerID *int64) ([]domain.Tweet, error) {
	items, err := HydrateItems(ctx, store, rows, viewerID)
	if err != nil {
		return nil, err
	}
	tweets := make([]domain.Tweet, len(items))
	for i, item := range items {
		tweets[i] = item.Tweet
	}
	return tweets, nil
}

// HydrateItems turns rows into feed items: tweets with authors and reply targets,
// plus the retweet (and retweeter) for rows that are retweets. It uses a fixed
// number of batch queries (no N+1).
func HydrateItems(ctx context.Context, store db.Querier, rows []Row, viewerID *int64) ([]domain.FeedItem, error) {
	if len(rows) == 0 {
		return []domain.FeedItem{}, nil
	}

	userIDs := make([]int64, 0, len(rows))
	var parentIDs []int64
	for _, r := range rows {
		userIDs = append(userIDs, r.Tweet.UserID)
		if r.RetweetedByID != nil {
			userIDs = append(userIDs, *r.RetweetedByID)
		}
		if r.Tweet.ParentID != nil {
			parentIDs = append(parentIDs, *r.Tweet.ParentID)
		}
	}

	// Only the parent's author is needed ("Replying to @username").
	parentAuthors := make(map[int64]int64, len(parentIDs))
	if len(parentIDs) > 0 {
		parents, err := store.GetTweetsByIDs(ctx, db.GetTweetsByIDsParams{TweetIds: uniq(parentIDs)})
		if err != nil {
			return nil, err
		}
		for _, p := range parents {
			parentAuthors[p.Tweet.ID] = p.Tweet.UserID
			userIDs = append(userIDs, p.Tweet.UserID)
		}
	}

	users, err := loadAuthors(ctx, store, uniq(userIDs), viewerID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.FeedItem, 0, len(rows))
	for _, r := range rows {
		t := domain.NewTweet(r.Tweet)
		t.Author = users[r.Tweet.UserID]
		t.IsLiked = r.IsLiked
		t.IsRetweeted = r.IsRetweeted
		t.IsFollowing = r.IsFollowing
		if r.Tweet.ParentID != nil {
			if authorID, ok := parentAuthors[*r.Tweet.ParentID]; ok {
				username := users[authorID].Username
				t.ParentUsername = &username
			}
		}
		item := domain.FeedItem{Tweet: t}
		if r.RetweetID != nil && r.RetweetedByID != nil && r.RetweetedAt != nil {
			item.Retweet = &domain.Retweet{ID: *r.RetweetID, By: users[*r.RetweetedByID], CreatedAt: *r.RetweetedAt}
		}
		result = append(result, item)
	}
	return result, nil
}

// Page trims a FetchLimit-sized result, hydrates it and derives the next cursor.
func Page(ctx context.Context, store db.Querier, rows []Row, limit int32, viewerID *int64, cursorOf func(Row) pagination.Cursor) (pagination.Page[domain.Tweet], error) {
	return page(ctx, store, rows, limit, viewerID, cursorOf, Hydrate)
}

// ItemPage is Page for timelines: the items keep the retweet that put a tweet there.
func ItemPage(ctx context.Context, store db.Querier, rows []Row, limit int32, viewerID *int64, cursorOf func(Row) pagination.Cursor) (pagination.Page[domain.FeedItem], error) {
	return page(ctx, store, rows, limit, viewerID, cursorOf, HydrateItems)
}

func page[T any](ctx context.Context, store db.Querier, rows []Row, limit int32, viewerID *int64, cursorOf func(Row) pagination.Cursor,
	hydrate func(context.Context, db.Querier, []Row, *int64) ([]T, error)) (pagination.Page[T], error) {
	rows, more := pagination.Trim(rows, limit)
	items, err := hydrate(ctx, store, rows, viewerID)
	if err != nil {
		return pagination.Page[T]{}, err
	}
	return pagination.Page[T]{Items: items, Next: pagination.NextFrom(rows, more, cursorOf)}, nil
}

// Rows converts any sqlc row type that shares Row's columns.
func Rows[T any](rows []T, toRow func(T) Row) []Row {
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i] = toRow(r)
	}
	return out
}

func loadAuthors(ctx context.Context, store db.Querier, ids []int64, viewerID *int64) (map[int64]domain.User, error) {
	rows, err := store.GetUsersByIDs(ctx, db.GetUsersByIDsParams{UserIds: ids, ViewerID: viewerID})
	if err != nil {
		return nil, err
	}
	authors := make(map[int64]domain.User, len(rows))
	for _, r := range rows {
		authors[r.User.ID] = domain.NewUser(r.User, r.IsFollowing)
	}
	return authors, nil
}

func uniq(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

var hashtagRegex = regexp.MustCompile(`(?i)(?:^|\s)#([a-z0-9_]+)`)

// extractHashtags returns unique, lowercased hashtags in order of appearance.
func extractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{}, len(matches))
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if _, dup := seen[tag]; dup {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}
