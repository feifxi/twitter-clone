package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/usecase"
)

func TestUserResponsesOnlyExposeEmailToOwner(t *testing.T) {
	user := usecase.UserItem{ID: 1, Username: "alice", Email: "alice@example.com"}

	public, _ := json.Marshal(newUserResponse(user))
	if strings.Contains(string(public), "alice@example.com") {
		t.Fatalf("public user response leaks email: %s", public)
	}

	own, _ := json.Marshal(newCurrentUserResponse(user))
	if !strings.Contains(string(own), `"email":"alice@example.com"`) || !strings.Contains(string(own), `"username":"alice"`) {
		t.Fatalf("current user response should be flat and include email: %s", own)
	}
}
