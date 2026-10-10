package httpapi

import (
	"testing"

	"github.com/chanombude/twitter-go-api/internal/pagination"
)

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
