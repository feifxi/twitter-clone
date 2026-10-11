// Package storage stores user media in S3 and maps object keys to public URLs.
// It trusts its inputs: callers own the key layout and must validate keys first.
package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const presignExpiry = 10 * time.Minute

type Config struct {
	Bucket string
	Region string
	// CDNDomain is the CloudFront domain (without scheme). When empty, public
	// URLs point straight at the bucket.
	CDNDomain string
}

type S3 struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	baseURL   string
}

// NewS3 returns a client whose operations fail when no bucket is configured,
// so the API can still boot locally without AWS credentials.
func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	if cfg.Bucket == "" {
		slog.WarnContext(ctx, "S3 bucket is not set; storage operations will fail")
		return &S3{}, nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	baseURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
	if cdn := strings.TrimRight(cfg.CDNDomain, "/"); cdn != "" {
		baseURL = "https://" + cdn
	}

	client := s3.NewFromConfig(awsCfg)
	return &S3{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
		baseURL:   baseURL,
	}, nil
}

// PresignUpload returns a PUT URL. contentType and contentLength are signed,
// so the client must upload exactly that type and size.
func (s *S3) PresignUpload(ctx context.Context, objectKey, contentType string, contentLength int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("S3 storage client is not configured")
	}

	out, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(objectKey),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	}, s3.WithPresignExpires(presignExpiry))
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}
	return out.URL, nil
}

// DeleteObject is best-effort: a leaked object is cheaper than failing the user's request.
func (s *S3) DeleteObject(ctx context.Context, objectKey string) error {
	if s.client == nil || objectKey == "" {
		return nil
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		slog.WarnContext(ctx, "failed to delete S3 object", "err", err, "key", objectKey)
	}
	return nil
}

func (s *S3) PublicURL(objectKey string) string {
	if s.baseURL == "" {
		return objectKey
	}
	return s.baseURL + "/" + objectKey
}

// KeyFromPublicURL reverses PublicURL. It returns false for URLs we don't host
// (e.g. a Google profile picture), which must never be deleted.
func (s *S3) KeyFromPublicURL(url string) (string, bool) {
	if s.baseURL == "" {
		return "", false
	}
	key, ok := strings.CutPrefix(url, s.baseURL+"/")
	if !ok || key == "" {
		return "", false
	}
	return key, true
}
