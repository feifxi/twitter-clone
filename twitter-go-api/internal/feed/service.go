// Package feed serves the home timelines and the profile timelines (posts,
// replies, media, likes).
package feed

import (
	"context"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/chanombude/twitter-go-api/internal/tweet"
)

type TweetPage = pagination.Page[domain.Tweet]

type Service struct {
	store db.Store
	now   func() time.Time
}

func New(store db.Store) *Service {
	return &Service{store: store, now: time.Now}
}

// Global is the ranked "For You" feed. The first page fixes an anchor time; later
// pages score against that same anchor so the ranking doesn't drift while scrolling.
func (s *Service) Global(ctx context.Context, req pagination.Request, viewerID *int64) (TweetPage, error) {
	anchor := s.now()
	var cursorScore *float64
	if req.After != nil {
		anchor = req.After.Anchor
		cursorScore = &req.After.Score
	}

	rows, err := s.store.ListForYouFeed(ctx, db.ListForYouFeedParams{
		ViewerID:    viewerID,
		Anchor:      anchor,
		CursorScore: cursorScore,
		CursorID:    req.AfterID(),
		PageLimit:   req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}

	rows, more := pagination.Trim(rows, req.Limit)
	items, err := tweet.Hydrate(ctx, s.store, tweet.Rows(rows, forYouRow), viewerID)
	if err != nil {
		return TweetPage{}, err
	}
	next := pagination.NextFrom(rows, more, func(r db.ListForYouFeedRow) pagination.Cursor {
		return pagination.Cursor{Anchor: anchor, Score: r.Score, ID: r.Tweet.ID}
	})
	return TweetPage{Items: items, Next: next}, nil
}

func forYouRow(r db.ListForYouFeedRow) tweet.Row {
	return tweet.Row{Tweet: r.Tweet, IsLiked: r.IsLiked, IsRetweeted: r.IsRetweeted, IsFollowing: r.IsFollowing}
}

// Following is the chronological feed of tweets and retweets by accounts userID follows.
func (s *Service) Following(ctx context.Context, userID int64, req pagination.Request) (TweetPage, error) {
	rows, err := s.store.ListFollowingTimeline(ctx, db.ListFollowingTimelineParams{
		ViewerID:   userID,
		CursorTime: req.AfterTime(),
		CursorKind: req.AfterKind(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}
	return tweet.Page(ctx, s.store, tweet.Rows(rows, func(r db.ListFollowingTimelineRow) tweet.Row { return tweet.Row(r) }), req.Limit, &userID, tweet.TimelineCursor)
}

// Posts is userID's profile timeline: their top-level tweets and retweets.
func (s *Service) Posts(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (TweetPage, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return TweetPage{}, err
	}
	rows, err := s.store.ListProfileTimeline(ctx, db.ListProfileTimelineParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorKind: req.AfterKind(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}
	return tweet.Page(ctx, s.store, tweet.Rows(rows, func(r db.ListProfileTimelineRow) tweet.Row { return tweet.Row(r) }), req.Limit, viewerID, tweet.TimelineCursor)
}

func (s *Service) Replies(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (TweetPage, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return TweetPage{}, err
	}
	rows, err := s.store.ListUserReplies(ctx, db.ListUserRepliesParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}
	return tweet.Page(ctx, s.store, tweet.Rows(rows, func(r db.ListUserRepliesRow) tweet.Row { return tweet.Row(r) }), req.Limit, viewerID, tweet.CreatedCursor)
}

func (s *Service) Media(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (TweetPage, error) {
	if err := s.requireUser(ctx, userID); err != nil {
		return TweetPage{}, err
	}
	rows, err := s.store.ListUserMedia(ctx, db.ListUserMediaParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}
	return tweet.Page(ctx, s.store, tweet.Rows(rows, func(r db.ListUserMediaRow) tweet.Row { return tweet.Row(r) }), req.Limit, viewerID, tweet.CreatedCursor)
}

// Likes lists tweets userID liked, most recent like first. Likes are private, as on X:
// only the user themselves may list them.
func (s *Service) Likes(ctx context.Context, userID int64, req pagination.Request, viewerID *int64) (TweetPage, error) {
	if viewerID == nil || *viewerID != userID {
		return TweetPage{}, apperr.Forbidden("likes are only visible to their owner")
	}
	rows, err := s.store.ListUserLikes(ctx, db.ListUserLikesParams{
		UserID:     userID,
		ViewerID:   viewerID,
		CursorTime: req.AfterTime(),
		CursorID:   req.AfterID(),
		PageLimit:  req.FetchLimit(),
	})
	if err != nil {
		return TweetPage{}, err
	}

	rows, more := pagination.Trim(rows, req.Limit)
	items, err := tweet.Hydrate(ctx, s.store, tweet.Rows(rows, func(r db.ListUserLikesRow) tweet.Row {
		return tweet.Row{Tweet: r.Tweet, IsLiked: r.IsLiked, IsRetweeted: r.IsRetweeted, IsFollowing: r.IsFollowing}
	}), viewerID)
	if err != nil {
		return TweetPage{}, err
	}
	next := pagination.NextFrom(rows, more, func(r db.ListUserLikesRow) pagination.Cursor {
		return pagination.Cursor{Time: r.LikedAt, ID: r.Tweet.ID}
	})
	return TweetPage{Items: items, Next: next}, nil
}

func (s *Service) requireUser(ctx context.Context, userID int64) error {
	_, err := s.store.GetUser(ctx, db.GetUserParams{ID: userID})
	return db.NotFound(err, "user")
}
