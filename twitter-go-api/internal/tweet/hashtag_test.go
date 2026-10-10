package tweet

import (
	"reflect"
	"testing"
)

func TestExtractHashtags(t *testing.T) {
	t.Parallel()

	got := extractHashtags("hello #Go #go #backend and #Go_Lang")
	want := []string{"go", "backend", "go_lang"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected hashtags: got %v want %v", got, want)
	}
}
