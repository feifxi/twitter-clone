package realtime_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chanombude/twitter-go-api/internal/realtime"
)

func TestHub_DeliversOnlyToTheTargetUser(t *testing.T) {
	hub := realtime.NewHub(nil)
	alice, unsubscribeAlice := hub.Subscribe(1)
	defer unsubscribeAlice()
	bob, unsubscribeBob := hub.Subscribe(2)
	defer unsubscribeBob()

	if err := hub.Publish(context.Background(), 1, "notification", map[string]int{"id": 99}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case ev := <-alice:
		var payload map[string]int
		if err := json.Unmarshal(ev.Payload, &payload); err != nil || payload["id"] != 99 || ev.Type != "notification" {
			t.Fatalf("unexpected event %+v (err %v)", ev, err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected alice to receive the event")
	}

	select {
	case ev := <-bob:
		t.Fatalf("bob should not receive alice's event, got %+v", ev)
	default:
	}
}

func TestHub_UnsubscribeClosesChannel(t *testing.T) {
	hub := realtime.NewHub(nil)
	events, unsubscribe := hub.Subscribe(1)
	unsubscribe()

	if _, open := <-events; open {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
	// Publishing to a user with no subscribers is a no-op.
	if err := hub.Publish(context.Background(), 1, "notification", struct{}{}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}
