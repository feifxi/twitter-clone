package httpapi

import "testing"

func TestBuildPageResponseTrimsAndSetsHasNext(t *testing.T) {
	t.Parallel()

	resp := BuildPageResponse([]int{1, 2, 3}, 2, 0)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0] != 1 || resp.Items[1] != 2 {
		t.Fatalf("unexpected items after trim: %+v", resp.Items)
	}
	if !resp.HasNext {
		t.Fatal("expected hasNext=true")
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatal("expected nextCursor to be set")
	}
}

func TestBuildPageResponseNoTrimAndNoNext(t *testing.T) {
	t.Parallel()

	resp := BuildPageResponse([]int{1, 2}, 2, 0)
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.HasNext {
		t.Fatal("expected hasNext=false")
	}
	if resp.NextCursor != nil {
		t.Fatal("expected nextCursor=nil when hasNext=false")
	}
}
