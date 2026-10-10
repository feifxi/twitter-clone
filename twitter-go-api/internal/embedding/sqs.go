// Package embedding queues tweets for vectorization (consumed by the embedding worker).
package embedding

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/rs/zerolog/log"
)

type message struct {
	TweetID int64  `json:"tweet_id"`
	Content string `json:"content"`
}

type Publisher interface {
	PublishEmbeddingEvent(ctx context.Context, tweetID int64, content string) error
}

type SQSPublisher struct {
	client   *sqs.Client
	queueURL string
}

// NewPublisher returns a no-op publisher when queueURL is empty (e.g. local dev).
func NewPublisher(ctx context.Context, queueURL, region string) (Publisher, error) {
	if queueURL == "" {
		log.Warn().Msg("SQS embedding queue URL is not set — embedding events will be dropped")
		return noopPublisher{}, nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config for SQS: %w", err)
	}
	return &SQSPublisher{client: sqs.NewFromConfig(awsCfg), queueURL: queueURL}, nil
}

func (p *SQSPublisher) PublishEmbeddingEvent(ctx context.Context, tweetID int64, content string) error {
	body, err := json.Marshal(message{TweetID: tweetID, Content: content})
	if err != nil {
		return fmt.Errorf("marshal embedding message: %w", err)
	}

	if _, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(body)),
	}); err != nil {
		return fmt.Errorf("send embedding message: %w", err)
	}
	return nil
}

type noopPublisher struct{}

func (noopPublisher) PublishEmbeddingEvent(context.Context, int64, string) error { return nil }
