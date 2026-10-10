package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/chanombude/twitter-go-api/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

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
// @Param			request	body			usecase.AssistantInput	true	"Chat query and history"
// @Success		200		{string}	string					"text/event-stream"
// @Security		BearerAuth
// @Failure		401		{object}	ErrorResponse
// @Router			/assistant [post]
func (server *Server) assistant(ctx *gin.Context) {
	var input usecase.AssistantInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		writeError(ctx, err)
		return
	}

	chunks, err := server.assistantUC.Chat(ctx.Request.Context(), input)
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("X-Accel-Buffering", "no")
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Writer.Flush()

	for text, err := range chunks {
		if err != nil {
			log.Error().Err(err).Str("request_id", middleware.GetRequestID(ctx)).Msg("assistant stream failed")
			writeSSE(ctx, "error", assistantStreamError{Message: "the assistant is unavailable, please try again"})
			return
		}
		writeSSE(ctx, "", assistantChunk{Text: text})
	}
	writeSSE(ctx, "done", struct{}{})
}

// writeSSE JSON-encodes data so newlines in the payload can't break SSE framing.
func writeSSE(ctx *gin.Context, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal SSE payload")
		return
	}
	if event != "" {
		_, _ = fmt.Fprintf(ctx.Writer, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(ctx.Writer, "data: %s\n\n", payload)
	ctx.Writer.Flush()
}
