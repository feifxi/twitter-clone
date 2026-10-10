// Package embedding turns tweets into vectors for the assistant's RAG search.
// Tweets are queued in embedding_jobs (in the same transaction that creates
// them) and a background Worker embeds them with Gemini.
package embedding

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/genai"
)

// Dimensions must match tweet_embeddings.embedding (vector(768)).
const Dimensions = 768

type TaskType string

const (
	// TaskDocument embeds content that will be searched (tweets).
	TaskDocument TaskType = "RETRIEVAL_DOCUMENT"
	// TaskQuery embeds a search query (the user's question).
	TaskQuery TaskType = "RETRIEVAL_QUERY"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Gemini embeds text with a Gemini embedding model.
type Gemini struct {
	client *genai.Client
	model  string
	task   TaskType
}

func NewGemini(client *genai.Client, model string, task TaskType) *Gemini {
	return &Gemini{client: client, model: model, task: task}
}

func (g *Gemini) Embed(ctx context.Context, text string) ([]float32, error) {
	dim := int32(Dimensions)
	res, err := g.client.Models.EmbedContent(ctx, g.model,
		[]*genai.Content{{Parts: []*genai.Part{{Text: text}}}},
		&genai.EmbedContentConfig{TaskType: string(g.task), OutputDimensionality: &dim},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini embed: %w", err)
	}
	if len(res.Embeddings) == 0 || len(res.Embeddings[0].Values) != Dimensions {
		return nil, errors.New("gemini embed: unexpected embedding shape")
	}
	return res.Embeddings[0].Values, nil
}
