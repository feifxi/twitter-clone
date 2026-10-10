package config

import "testing"

func TestSplitList_Empty(t *testing.T) {
	t.Parallel()

	if got := splitList(""); got != nil {
		t.Fatalf("expected nil for empty string, got %v", got)
	}
}

func TestSplitList_Single(t *testing.T) {
	t.Parallel()

	got := splitList("https://example.com")
	if len(got) != 1 || got[0] != "https://example.com" {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestSplitList_MultipleWithSpaces(t *testing.T) {
	t.Parallel()

	got := splitList("https://a.com , https://b.com,  https://c.com")
	if len(got) != 3 {
		t.Fatalf("expected 3 origins, got %d: %v", len(got), got)
	}
	if got[1] != "https://b.com" {
		t.Fatalf("unexpected middle origin: %q", got[1])
	}
}

func TestSplitList_SkipsEmptySegments(t *testing.T) {
	t.Parallel()

	got := splitList("https://a.com, , https://b.com")
	if len(got) != 2 {
		t.Fatalf("expected 2 origins after skipping empty, got %d: %v", len(got), got)
	}
}
