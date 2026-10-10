// Package pagination implements opaque, keyset-style cursors. A cursor records
// the sort key of the last item a client has seen, so the next page starts
// right after it no matter how many rows were inserted meanwhile (unlike OFFSET).
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

const (
	DefaultLimit int32 = 20
	MaxLimit     int32 = 50
)

var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor holds whichever sort key the endpoint orders by; unused fields stay zero.
type Cursor struct {
	Time   time.Time `json:"t,omitzero"`
	ID     int64     `json:"i,omitempty"`
	Kind   int32     `json:"k,omitempty"` // timelines mixing tweets (0) and retweets (1)
	Score  float64   `json:"s,omitempty"` // ranked feeds
	Anchor time.Time `json:"a,omitzero"`  // ranked feeds: the time scores were computed at
	Offset int32     `json:"o,omitempty"` // relevance-ranked lists (search)
}

// Request asks for the page after After (nil = first page).
type Request struct {
	After *Cursor
	Limit int32
}

type Page[T any] struct {
	Items []T
	Next  *Cursor // nil on the last page
}

// FetchLimit is the row count to query: one extra row reveals whether another page exists.
func (r Request) FetchLimit() int32 { return r.Limit + 1 }

// Offset is the row offset for offset-paged (relevance-ranked) lists.
func (r Request) Offset() int32 {
	if r.After == nil {
		return 0
	}
	return r.After.Offset
}

// NextOffset is the cursor for the page after an offset-paged request.
func (r Request) NextOffset() Cursor {
	return Cursor{Offset: r.Offset() + r.Limit}
}

// AfterTime / AfterID / AfterKind feed sqlc's nullable cursor parameters.
func (r Request) AfterTime() *time.Time {
	if r.After == nil {
		return nil
	}
	t := r.After.Time
	return &t
}

func (r Request) AfterID() *int64 {
	if r.After == nil {
		return nil
	}
	id := r.After.ID
	return &id
}

func (r Request) AfterKind() *int32 {
	if r.After == nil {
		return nil
	}
	k := r.After.Kind
	return &k
}

// Trim drops the probe row fetched by FetchLimit and reports whether more rows exist.
func Trim[T any](rows []T, limit int32) ([]T, bool) {
	if int32(len(rows)) <= limit {
		return rows, false
	}
	return rows[:limit], true
}

// NextFrom builds the next cursor from the last kept row, or returns nil when there are no more rows.
func NextFrom[T any](rows []T, more bool, cursorOf func(T) Cursor) *Cursor {
	if !more || len(rows) == 0 {
		return nil
	}
	c := cursorOf(rows[len(rows)-1])
	return &c
}

func Encode(c Cursor) string {
	data, _ := json.Marshal(c) // Cursor always marshals
	return base64.RawURLEncoding.EncodeToString(data)
}

func Decode(s string) (*Cursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil || c.Offset < 0 {
		return nil, ErrInvalidCursor
	}
	return &c, nil
}
