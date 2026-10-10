package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/logger"
)

func TestLoggerAddsRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, true, slog.LevelInfo)

	ctx := logger.WithRequestID(context.Background(), "req-123")
	log.InfoContext(ctx, "hello")
	log.Info("no context")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %q", buf.String())
	}
	if !strings.Contains(lines[0], `"request_id":"req-123"`) {
		t.Fatalf("expected request_id on contextual log, got %s", lines[0])
	}
	if strings.Contains(lines[1], "request_id") {
		t.Fatalf("expected no request_id without context, got %s", lines[1])
	}
}
