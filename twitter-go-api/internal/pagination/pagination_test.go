package pagination_test

import (
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/pagination"
)

func TestCursorRoundTrip(t *testing.T) {
	in := pagination.Cursor{
		Time:   time.Date(2026, 10, 10, 12, 0, 0, 123456000, time.UTC),
		ID:     42,
		Kind:   1,
		Score:  0.123456789012345,
		Anchor: time.Date(2026, 10, 10, 13, 0, 0, 0, time.UTC),
	}

	out, err := pagination.Decode(pagination.Encode(in))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Time.Equal(in.Time) || !out.Anchor.Equal(in.Anchor) || out.ID != in.ID || out.Kind != in.Kind || out.Score != in.Score {
		t.Fatalf("round trip mismatch: got %+v want %+v", *out, in)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	for _, s := range []string{"%%%", "bm90LWpzb24", pagination.Encode(pagination.Cursor{Offset: -5})} {
		if _, err := pagination.Decode(s); err == nil {
			t.Fatalf("expected %q to be rejected", s)
		}
	}
}

func TestTrimAndNext(t *testing.T) {
	rows, more := pagination.Trim([]int{1, 2, 3}, 2)
	if len(rows) != 2 || !more {
		t.Fatalf("expected 2 rows and more, got %v %v", rows, more)
	}
	next := pagination.NextFrom(rows, more, func(r int) pagination.Cursor { return pagination.Cursor{ID: int64(r)} })
	if next == nil || next.ID != 2 {
		t.Fatalf("expected cursor from last kept row, got %+v", next)
	}

	rows, more = pagination.Trim([]int{1, 2}, 2)
	if more || pagination.NextFrom(rows, more, func(int) pagination.Cursor { return pagination.Cursor{} }) != nil {
		t.Fatal("expected no next page")
	}
}

func TestOffsetRequests(t *testing.T) {
	first := pagination.Request{Limit: 20}
	if first.Offset() != 0 || first.NextOffset().Offset != 20 {
		t.Fatalf("unexpected first page offsets")
	}
	second := pagination.Request{After: &pagination.Cursor{Offset: 20}, Limit: 20}
	if second.Offset() != 20 || second.NextOffset().Offset != 40 {
		t.Fatalf("unexpected second page offsets")
	}
}
