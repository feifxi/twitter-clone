package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chanombude/twitter-go-api/internal/storage"
)

func TestNewS3_URLs(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	cases := map[string]struct {
		cfg        storage.Config
		publicURL  string
		presignURL string
	}{
		"aws": {
			cfg:        storage.Config{Bucket: "media", Region: "ap-southeast-1"},
			publicURL:  "https://media.s3.ap-southeast-1.amazonaws.com/tweets/a.png",
			presignURL: "https://media.s3.ap-southeast-1.amazonaws.com/tweets/a.png?",
		},
		"cdn": {
			cfg:        storage.Config{Bucket: "media", Region: "ap-southeast-1", CDNDomain: "cdn.example.com"},
			publicURL:  "https://cdn.example.com/tweets/a.png",
			presignURL: "https://media.s3.ap-southeast-1.amazonaws.com/tweets/a.png?",
		},
		"local endpoint": {
			cfg:        storage.Config{Bucket: "media", Region: "ap-southeast-1", Endpoint: "http://localhost:9000/"},
			publicURL:  "http://localhost:9000/media/tweets/a.png",
			presignURL: "http://localhost:9000/media/tweets/a.png?",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, err := storage.NewS3(context.Background(), tc.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := s.PublicURL("tweets/a.png"); got != tc.publicURL {
				t.Errorf("PublicURL = %q, want %q", got, tc.publicURL)
			}
			if key, ok := s.KeyFromPublicURL(tc.publicURL); !ok || key != "tweets/a.png" {
				t.Errorf("KeyFromPublicURL = %q, %v", key, ok)
			}
			url, err := s.PresignUpload(context.Background(), "tweets/a.png", "image/png", 10)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(url, tc.presignURL) {
				t.Errorf("presigned URL %q, want prefix %q", url, tc.presignURL)
			}
		})
	}
}
