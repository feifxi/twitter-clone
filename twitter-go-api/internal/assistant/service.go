// Package assistant streams Gemini chat answers grounded in recent tweets (RAG).
package assistant

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

type Config struct {
	ChatModel      string
	EmbeddingModel string
	EnableRAG      bool
}

type Role string

const (
	RoleUser  Role = "user"
	RoleModel Role = "model"
)

type HistoryItem struct {
	Role Role
	Text string
}

type Input struct {
	Query   string
	History []HistoryItem
}

type Service struct {
	cfg   Config
	store db.Store
	genai *genai.Client
}

// New accepts a nil client; the assistant then reports itself as not configured.
func New(cfg Config, store db.Store, client *genai.Client) *Service {
	return &Service{cfg: cfg, store: store, genai: client}
}

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
func (s *Service) Chat(ctx context.Context, input Input) (iter.Seq2[string, error], error) {
	if s.genai == nil {
		return nil, apperr.E(apperr.KindInternal, "assistant is not configured", fmt.Errorf("GEMINI_API_KEY is not set"))
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{
			Text: assistantSystemPreamble + "\n\n[Timeline Context]\n" + s.timelineContext(ctx, input.Query),
		}}},
	}
	contents := buildChatContents(input)

	return func(yield func(string, error) bool) {
		for resp, err := range s.genai.Models.GenerateContentStream(ctx, s.cfg.ChatModel, contents, config) {
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
func (s *Service) timelineContext(ctx context.Context, query string) string {
	if !s.cfg.EnableRAG {
		return "Timeline context is currently disabled."
	}

	dim := int32(ragEmbeddingDimensions)
	embedRes, err := s.genai.Models.EmbedContent(ctx, s.cfg.EmbeddingModel,
		[]*genai.Content{{Parts: []*genai.Part{{Text: query}}}},
		&genai.EmbedContentConfig{TaskType: "RETRIEVAL_QUERY", OutputDimensionality: &dim},
	)
	if err != nil || len(embedRes.Embeddings) == 0 {
		log.Error().Err(err).Msg("failed to vectorize query for RAG, falling back to base chat")
		return "Timeline context unavailable (vectorization failed)."
	}

	queryVector := pgvector.NewVector(embedRes.Embeddings[0].Values)
	rows, err := s.store.ListRelatedTweetsByEmbedding(ctx, db.ListRelatedTweetsByEmbeddingParams{
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

func buildChatContents(input Input) []*genai.Content {
	history := input.History
	if len(history) > assistantMaxHistory {
		history = history[len(history)-assistantMaxHistory:]
	}

	contents := make([]*genai.Content, 0, len(history)+1)
	for _, h := range history {
		role := genai.RoleUser
		if h.Role == RoleModel {
			role = genai.RoleModel
		}
		contents = append(contents, &genai.Content{Role: role, Parts: []*genai.Part{{Text: h.Text}}})
	}
	return append(contents, &genai.Content{Role: genai.RoleUser, Parts: []*genai.Part{{Text: input.Query}}})
}
