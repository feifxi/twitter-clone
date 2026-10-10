package search

import "testing"

func TestBuildTSQuery(t *testing.T) {
	t.Parallel()

	got := buildTSQuery("  hello, world!! go-lang ")
	want := "hello & world & go & lang"
	if got != want {
		t.Fatalf("unexpected tsquery: got %q want %q", got, want)
	}
}
