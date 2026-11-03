package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/pagination"
)

func TestGetGlobalFeed_UsecaseErrorPropagates(t *testing.T) {
	t.Parallel()

	mock := &fakeFeeds{
		globalFn: func(context.Context, pagination.Request, *int64) (pagination.Page[domain.FeedItem], error) {
			return pagination.Page[domain.FeedItem]{}, apperr.Internal("feed failure", errors.New("db error"))
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global", nil, "")
	s := &Server{svc: Services{Feeds: mock}}
	s.getGlobalFeed(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for feed error, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetGlobalFeedPassesPageRequestAndEncodesNextCursor(t *testing.T) {
	t.Parallel()

	var got pagination.Request
	fake := &fakeFeeds{globalFn: func(_ context.Context, req pagination.Request, _ *int64) (pagination.Page[domain.FeedItem], error) {
		got = req
		return pagination.Page[domain.FeedItem]{
			Items: []domain.FeedItem{{Tweet: domain.Tweet{ID: 1, CreatedAt: time.Now(), Author: domain.User{ID: 11, Email: "u1@example.com"}}}},
			Next:  &pagination.Cursor{ID: 1},
		}, nil
	}}

	cursor := pagination.Encode(pagination.Cursor{ID: 5})
	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global?size=2&cursor="+cursor, nil, "")
	(&Server{svc: Services{Feeds: fake}}).getGlobalFeed(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if got.Limit != 2 || got.After == nil || got.After.ID != 5 {
		t.Fatalf("unexpected page request: %+v", got)
	}

	var body PageResponse[FeedItemResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(body.Items) != 1 || !body.HasNext || body.NextCursor == nil || body.Items[0].Kind != "tweet" || body.Items[0].Tweet.ID != 1 {
		t.Fatalf("unexpected page: %+v", body)
	}
	if next, err := pagination.Decode(*body.NextCursor); err != nil || next.ID != 1 {
		t.Fatalf("next cursor does not round-trip: %v %+v", err, next)
	}
	if strings.Contains(rec.Body.String(), "u1@example.com") {
		t.Fatal("feed must not leak author emails")
	}
}

// The feed item contract: every field present (nulls included), retweets wrap the
// original tweet, and the last page says nextCursor: null.
func TestFeedItemsAndLastPageShape(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC)
	fake := &fakeFeeds{globalFn: func(context.Context, pagination.Request, *int64) (pagination.Page[domain.FeedItem], error) {
		return pagination.Page[domain.FeedItem]{Items: []domain.FeedItem{
			{Tweet: domain.Tweet{ID: 1, CreatedAt: at, Author: domain.User{ID: 11, Username: "author"}}},
			{Tweet: domain.Tweet{ID: 2, CreatedAt: at, Author: domain.User{ID: 11, Username: "author"}},
				Retweet: &domain.Retweet{ID: 7, By: domain.User{ID: 12, Username: "fan"}, CreatedAt: at}},
		}}, nil
	}}
	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global", nil, "")
	(&Server{svc: Services{Feeds: fake}}).getGlobalFeed(ctx)

	var body struct {
		Items      []map[string]json.RawMessage `json:"items"`
		HasNext    bool                         `json:"hasNext"`
		NextCursor *string                      `json:"nextCursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !strings.Contains(rec.Body.String(), `"nextCursor":null`) || body.HasNext {
		t.Fatalf("the last page must say nextCursor: null, got %s", rec.Body.String())
	}
	if string(body.Items[0]["kind"]) != `"tweet"` || string(body.Items[0]["retweet"]) != "null" {
		t.Fatalf("plain tweet item: %s", rec.Body.String())
	}
	var rt RetweetResponse
	if string(body.Items[1]["kind"]) != `"retweet"` || json.Unmarshal(body.Items[1]["retweet"], &rt) != nil ||
		rt.ID != 7 || rt.Retweeter.Username != "fan" || !rt.CreatedAt.Equal(at) {
		t.Fatalf("retweet item: %s", rec.Body.String())
	}
	if strings.Contains(string(body.Items[1]["tweet"]), "retweetedBy") {
		t.Fatal("the tweet inside an item is the plain tweet shape")
	}
}
