package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/domain"
)

func TestSearchHashtags_ReturnsResults(t *testing.T) {
	t.Parallel()

	mock := &fakeSearch{
		hashtagsFn: func(_ context.Context, _ string, _ int32) ([]domain.Hashtag, error) {
			return []domain.Hashtag{{Text: "golang"}}, nil
		},
	}

	ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/search/hashtags?q=go", nil, "")
	s := &Server{svc: Services{Search: mock}}
	s.searchHashtags(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v body=%s", err, rec.Body.String())
	}
	if len(out) != 1 || out[0]["text"] != "golang" {
		t.Fatalf("unexpected response: %v", out)
	}
}
