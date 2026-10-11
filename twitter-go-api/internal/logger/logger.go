// Package logger configures the process-wide slog logger and carries the
// request ID through context so every log line inside a request is correlated.
package logger

import (
	"context"
	"io"
	"log/slog"
)

type ctxKey struct{}

// New returns a JSON logger for production and a human-readable one otherwise.
// Records logged with a context that carries a request ID get a request_id attribute.
func New(w io.Writer, production bool, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler = slog.NewTextHandler(w, opts)
	if production {
		handler = slog.NewJSONHandler(w, opts)
	}
	return slog.New(requestIDHandler{handler})
}

// WithRequestID returns a context whose log records will include id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestID returns the request ID stored by WithRequestID, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

type requestIDHandler struct {
	slog.Handler
}

func (h requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{h.Handler.WithAttrs(attrs)}
}

func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{h.Handler.WithGroup(name)}
}
