// Package realtime fans events out to a user's open streams. With Redis, events
// published on any API instance reach subscribers on every instance; without
// it, delivery is in-process only (fine for a single instance).
package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	redisChannel     = "realtime:events"
	subscriberBuffer = 32
)

type Event struct {
	UserID  int64           `json:"userId"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Hub struct {
	rdb  *redis.Client
	mu   sync.RWMutex
	subs map[int64]map[chan Event]struct{}
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{rdb: rdb, subs: make(map[int64]map[chan Event]struct{})}
}

// Run relays events from Redis to local subscribers until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	if h.rdb == nil {
		return
	}

	pubsub := h.rdb.Subscribe(ctx, redisChannel)
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var ev Event
		if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
			slog.ErrorContext(ctx, "realtime: malformed event from Redis", "err", err)
			continue
		}
		h.deliver(ev)
	}
}

// Publish sends payload, JSON-encoded, to every stream userID has open.
func (h *Hub) Publish(ctx context.Context, userID int64, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", eventType, err)
	}
	ev := Event{UserID: userID, Type: eventType, Payload: data}

	if h.rdb == nil {
		h.deliver(ev)
		return nil
	}

	msg, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := h.rdb.Publish(ctx, redisChannel, msg).Err(); err != nil {
		// Users connected to this instance still get the event.
		h.deliver(ev)
		return fmt.Errorf("publish to redis: %w", err)
	}
	return nil
}

// Subscribe returns a channel of userID's events and a function that must be
// called to unsubscribe (it also closes the channel).
func (h *Hub) Subscribe(userID int64) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)

	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan Event]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs[userID], ch)
		if len(h.subs[userID]) == 0 {
			delete(h.subs, userID)
		}
		close(ch)
	}
}

// deliver never blocks the publisher: a slow client drops events rather than stalling everyone.
func (h *Hub) deliver(ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[ev.UserID] {
		select {
		case ch <- ev:
		default:
			slog.Warn("realtime: dropped event, subscriber buffer full", "user_id", ev.UserID, "type", ev.Type)
		}
	}
}
