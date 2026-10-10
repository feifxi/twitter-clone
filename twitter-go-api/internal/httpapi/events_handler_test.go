package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/domain"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/realtime"
)

func TestWriteRealtimeEventMapsToDTOs(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		evType   string
		payload  any
		wantName string
		wantJSON string
	}{
		"notification": {
			evType:   notification.EventType,
			payload:  domain.Notification{ID: 5, Type: domain.NotificationLike, Actor: domain.User{ID: 2, Username: "bob", Email: "bob@example.com"}},
			wantName: "event: notification",
			wantJSON: `"type":"LIKE"`,
		},
		"direct message": {
			evType:   message.EventType,
			payload:  domain.Message{ID: 9, ConversationID: 3, Content: "hi", Sender: domain.User{ID: 2, Email: "bob@example.com"}},
			wantName: "event: dm",
			wantJSON: `"conversationId":3`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, _ := json.Marshal(tc.payload)
			c, rec := newHandlerTestContext(http.MethodGet, "/api/v1/events", nil, "")

			writeRealtimeEvent(c, realtime.Event{UserID: 1, Type: tc.evType, Payload: data})

			body := rec.Body.String()
			if !strings.Contains(body, tc.wantName) || !strings.Contains(body, tc.wantJSON) {
				t.Fatalf("unexpected SSE frame: %q", body)
			}
			if strings.Contains(body, "bob@example.com") {
				t.Fatalf("SSE frame leaks email: %q", body)
			}
		})
	}
}
