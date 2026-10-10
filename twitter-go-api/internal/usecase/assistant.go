package usecase

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/chanombude/twitter-go-api/internal/apperr"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/pgvector/pgvector-go"
	"github.com/rs/zerolog/log"
	"google.golang.org/genai"
)

const (
	assistantMaxHistory     = 30
	ragResultLimit          = 5
	ragMaxCosineDistance    = 0.5
	ragEmbeddingDimensions  = 768
	assistantSystemPreamble = `You are the helpful, professional, and polite AI assistant for "Chanom Twitter", a social media platform and Twitter clone.
Your goal is to provide accurate, concise, and friendly information to users of Chanom Twitter.
You have access to the user's recent [Chat History] and [Timeline Context] from their feed.
Prioritize the Timeline Context when answering questions about the app's content or the user's recent experience.
Maintain conversational continuity and a helpful tone consistent with a premium social experience.
CRITICAL: Your memory is limited to the provided history. If asked about older context you don't have, politely state you don't have access to that part of the conversation and ask for clarification.`
)

// NewGeminiClient returns nil when no API key is configured; the assistant then reports itself unavailable.
func NewGeminiClient(ctx context.Context, apiKey string) (*genai.Client, error) {
	if apiKey == "" {
		return nil, nil
	}
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
}

// Chat streams the model's reply as text chunks. Errors before streaming starts are returned
// directly; errors mid-stream are yielded so the caller can report them on the open stream.
func (u *AssistantUsecase) Chat(ctx context.Context, input AssistantInput) (iter.Seq2[string, error], error) {
	if u.genai == nil {
		return nil, apperr.E(apperr.KindInternal, "assistant is not configured", fmt.Errorf("GEMINI_API_KEY is not set"))
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{
			Text: assistantSystemPreamble + "\n\n[Timeline Context]\n" + u.timelineContext(ctx, input.Query),
		}}},
	}
	contents := buildChatContents(input)

	return func(yield func(string, error) bool) {
		for resp, err := range u.genai.Models.GenerateContentStream(ctx, u.config.GeminiChatModel, contents, config) {
			if err != nil {
				yield("", err)
				return
			}
			if text := resp.Text(); text != "" {
				if !yield(text, nil) {
					return
				}
			}
		}
	}, nil
}

// timelineContext never fails the chat: RAG problems degrade to a context-less answer.
func (u *AssistantUsecase) timelineContext(ctx context.Context, query string) string {
	if !u.config.EnableRAG {
		return "Timeline context is currently disabled."
	}

	dim := int32(ragEmbeddingDimensions)
	embedRes, err := u.genai.Models.EmbedContent(ctx, u.config.GeminiEmbeddingModel,
		[]*genai.Content{{Parts: []*genai.Part{{Text: query}}}},
		&genai.EmbedContentConfig{TaskType: "RETRIEVAL_QUERY", OutputDimensionality: &dim},
	)
	if err != nil || len(embedRes.Embeddings) == 0 {
		log.Error().Err(err).Msg("failed to vectorize query for RAG, falling back to base chat")
		return "Timeline context unavailable (vectorization failed)."
	}

	queryVector := pgvector.NewVector(embedRes.Embeddings[0].Values)
	rows, err := u.store.ListRelatedTweetsByEmbedding(ctx, db.ListRelatedTweetsByEmbeddingParams{
		QueryEmbedding: &queryVector,
		MaxDistance:    ragMaxCosineDistance,
		ResultLimit:    ragResultLimit,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve context from DB for RAG")
		return "Timeline context unavailable (DB error)."
	}
	if len(rows) == 0 {
		return "No recent matching posts found in the timeline."
	}

	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, "- "+row.Content)
	}
	return strings.Join(parts, "\n")
}

func buildChatContents(input AssistantInput) []*genai.Content {
	history := input.History
	if len(history) > assistantMaxHistory {
		history = history[len(history)-assistantMaxHistory:]
	}

	contents := make([]*genai.Content, 0, len(history)+1)
	for _, h := range history {
		role := genai.RoleUser
		if h.Role == "assistant" || h.Role == "model" {
			role = genai.RoleModel
		}
		contents = append(contents, &genai.Content{Role: role, Parts: []*genai.Part{{Text: h.Text}}})
	}
	return append(contents, &genai.Content{Role: genai.RoleUser, Parts: []*genai.Part{{Text: input.Query}}})
}
