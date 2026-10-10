package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/assistant"
	"github.com/gin-gonic/gin"
)

type assistantHistoryItem struct {
	Role string `json:"role" binding:"required,oneof=user model assistant"`
	Text string `json:"text"`
}

type assistantRequest struct {
	Query   string                 `json:"query" binding:"required,max=4000"`
	History []assistantHistoryItem `json:"history" binding:"max=100,dive"`
}

func (r assistantRequest) toInput() assistant.Input {
	history := make([]assistant.HistoryItem, 0, len(r.History))
	for _, h := range r.History {
		role := assistant.RoleUser
		if h.Role == "model" || h.Role == "assistant" {
			role = assistant.RoleModel
		}
		history = append(history, assistant.HistoryItem{Role: role, Text: h.Text})
	}
	return assistant.Input{Query: r.Query, History: history}
}

type assistantChunk struct {
	Text string `json:"text"`
}

type assistantStreamError struct {
	Message string `json:"message"`
}

// assistant godoc
// @Summary		AI Assistant Chat (SSE)
// @Description	Interact with the Gemini-powered AI assistant. Streams Server-Sent Events: each `data:` line is a JSON `{"text": "..."}` chunk; the stream ends with `event: done`, or `event: error` with `{"message": "..."}`.
// @Tags			Assistant
// @Accept			json
// @Produce			text/event-stream
// @Param			request	body			assistantRequest	true	"Chat query and history"
// @Success		200		{string}	string					"text/event-stream"
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/assistant [post]
func (s *Server) assistant(c *gin.Context) {
	var req assistantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, err)
		return
	}

	chunks, err := s.svc.Assistant.Chat(c.Request.Context(), req.toInput())
	if err != nil {
		writeError(c, err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	for text, err := range chunks {
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "assistant stream failed", "err", err)
			writeSSE(c, "error", assistantStreamError{Message: "the assistant is unavailable, please try again"})
			return
		}
		writeSSE(c, "", assistantChunk{Text: text})
	}
	writeSSE(c, "done", struct{}{})
}

// writeSSE JSON-encodes data so newlines in the payload can't break SSE framing.
func writeSSE(c *gin.Context, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to marshal SSE payload", "err", err)
		return
	}
	if event != "" {
		_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	c.Writer.Flush()
}
