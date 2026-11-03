package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/pagination"
	"github.com/gin-gonic/gin"
)

func TestPageRequestValidation(t *testing.T) {
	t.Parallel()

	for _, query := range []string{"?cursor=not-a-cursor", "?size=0", "?size=abc"} {
		ctx, rec := newHandlerTestContext(http.MethodGet, "/api/v1/feeds/global"+query, nil, "")
		(&Server{svc: Services{Feeds: &fakeFeeds{}}}).getGlobalFeed(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", query, rec.Code)
		}
	}

	ctx, _ := newHandlerTestContext(http.MethodGet, "/x?size=500", nil, "")
	req, ok := parsePageRequest(ctx)
	if !ok || req.Limit != pagination.MaxLimit {
		t.Fatalf("expected size to be capped at %d, got %+v", pagination.MaxLimit, req)
	}
}

func TestParseLimit_DefaultWhenAbsent(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10, got %d", got)
	}
}

func TestParseLimit_ParsesValidValue(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=25", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
}

func TestParseLimit_CapsAtMax(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=999", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 50 {
		t.Fatalf("expected cap at 50, got %d", got)
	}
}

func TestParseLimit_InvalidFallsBackToDefault(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10 on invalid input, got %d", got)
	}
}

func TestParseLimit_ZeroFallsBackToDefault(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?limit=0", nil)

	got := parseLimit(ctx, 10, 50)
	if got != 10 {
		t.Fatalf("expected default 10 for limit=0, got %d", got)
	}
}

func TestNewPageResponse(t *testing.T) {
	t.Parallel()

	last := newPageResponse(pagination.Page[int]{Items: []int{1, 2}}, func(i int) int { return i * 10 })
	if last.HasNext || last.NextCursor != nil || len(last.Items) != 2 || last.Items[1] != 20 {
		t.Fatalf("unexpected last page: %+v", last)
	}

	more := newPageResponse(pagination.Page[int]{Items: []int{}, Next: &pagination.Cursor{ID: 7}}, func(i int) int { return i })
	if !more.HasNext || more.NextCursor == nil || more.Items == nil {
		t.Fatalf("unexpected page with more: %+v", more)
	}
}
