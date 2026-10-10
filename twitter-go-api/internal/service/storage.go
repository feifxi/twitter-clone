package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/rs/zerolog/log"
)

const presignExpiry = 10 * time.Minute

// StorageService abstracts object storage. Callers own the object key layout and
// must validate keys before passing them in; this layer trusts its inputs.
type StorageService interface {
	// PresignUpload returns a PUT URL. contentType and contentLength are signed,
	// so the client must upload exactly that type and size.
	PresignUpload(ctx context.Context, objectKey, contentType string, contentLength int64) (string, error)
	DeleteObject(ctx context.Context, objectKey string) error
	PublicURL(objectKey string) string
	// KeyFromPublicURL reverses PublicURL. It returns false for URLs we don't host
	// (e.g. a Google profile picture), which must never be deleted.
	KeyFromPublicURL(url string) (string, bool)
}

type S3StorageService struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucketName string
	baseURL    string
}

func NewS3StorageService(cfg config.Config) (StorageService, error) {
	if cfg.S3BucketName == "" {
		log.Warn().Msg("S3_BUCKET_NAME is not set — storage operations will fail")
		return &S3StorageService{}, nil
	}

	region := cfg.S3Region
	if region == "" {
		region = "ap-southeast-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	baseURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.S3BucketName, region)
	if cdn := strings.TrimRight(cfg.CloudFrontDomain, "/"); cdn != "" {
		baseURL = "https://" + cdn
	}

	client := s3.NewFromConfig(awsCfg)
	return &S3StorageService{
		client:     client,
		presigner:  s3.NewPresignClient(client),
		bucketName: cfg.S3BucketName,
		baseURL:    baseURL,
	}, nil
}

func (s *S3StorageService) PresignUpload(ctx context.Context, objectKey, contentType string, contentLength int64) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("S3 storage client is not configured")
	}

	out, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(objectKey),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	}, s3.WithPresignExpires(presignExpiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return out.URL, nil
}

// DeleteObject is best-effort: a leaked object is cheaper than failing the user's request.
func (s *S3StorageService) DeleteObject(ctx context.Context, objectKey string) error {
	if s.client == nil || objectKey == "" {
		return nil
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Warn().Err(err).Str("key", objectKey).Msg("Failed to delete S3 object")
	}
	return nil
}

func (s *S3StorageService) PublicURL(objectKey string) string {
	if s.baseURL == "" {
		return objectKey
	}
	return s.baseURL + "/" + objectKey
}

func (s *S3StorageService) KeyFromPublicURL(url string) (string, bool) {
	if s.baseURL == "" {
		return "", false
	}
	key, ok := strings.CutPrefix(url, s.baseURL+"/")
	if !ok || key == "" {
		return "", false
	}
	return key, true
}
