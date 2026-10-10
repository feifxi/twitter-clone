package tweet

import (
	"context"
	"regexp"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
)

// Row is the shape shared by every tweet-listing query (tweet + viewer flags).
// Each sqlc row type converts to it directly: Row(r).
type Row struct {
	Tweet       db.Tweet
	IsLiked     bool
	IsRetweeted bool
	IsFollowing bool
}

// Hydrate turns query rows into domain tweets with authors, reply targets and
// retweeted originals filled in, using a fixed number of batch queries (no N+1).
func Hydrate[T any](ctx context.Context, store db.Querier, rows []T, viewerID *int64, toRow func(T) Row) ([]domain.Tweet, error) {
	if len(rows) == 0 {
		return []domain.Tweet{}, nil
	}

	base := make([]Row, len(rows))
	userIDs := make([]int64, 0, len(rows))
	var refIDs []int64
	for i, r := range rows {
		base[i] = toRow(r)
		t := base[i].Tweet
		userIDs = append(userIDs, t.UserID)
		if t.ParentID != nil {
			refIDs = append(refIDs, *t.ParentID)
		}
		if t.RetweetID != nil {
			refIDs = append(refIDs, *t.RetweetID)
		}
	}

	// Referenced tweets (parents and retweeted originals) are loaded one level deep.
	refs := make(map[int64]Row)
	if len(refIDs) > 0 {
		refRows, err := store.GetTweetsByIDs(ctx, db.GetTweetsByIDsParams{TweetIds: uniq(refIDs), ViewerID: viewerID})
		if err != nil {
			return nil, err
		}
		for _, r := range refRows {
			refs[r.Tweet.ID] = Row(r)
			userIDs = append(userIDs, r.Tweet.UserID)
		}
	}

	authors, err := loadAuthors(ctx, store, uniq(userIDs), viewerID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Tweet, 0, len(base))
	for _, r := range base {
		item := toDomain(r, authors)
		if r.Tweet.ParentID != nil {
			if parent, ok := refs[*r.Tweet.ParentID]; ok {
				username := authors[parent.Tweet.UserID].Username
				item.ParentUsername = &username
			}
		}
		if r.Tweet.RetweetID != nil {
			if original, ok := refs[*r.Tweet.RetweetID]; ok {
				o := toDomain(original, authors)
				item.OriginalTweet = &o
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func toDomain(r Row, authors map[int64]domain.User) domain.Tweet {
	t := domain.NewTweet(r.Tweet)
	t.Author = authors[r.Tweet.UserID]
	t.IsLiked = r.IsLiked
	t.IsRetweeted = r.IsRetweeted
	t.IsFollowing = r.IsFollowing
	return t
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
