package config

import "testing"

func TestValidateForRuntime_ProductionRequiresFrontendURL(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Environment:       "production",
		DBSource:          "postgresql://host/db?sslmode=require",
		S3BucketName:      "my-bucket",
		S3Region:          "ap-southeast-1",
		CloudFrontDomain:  "d123.cloudfront.net",
		GoogleClientID:    "client-id",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	if err := cfg.ValidateForRuntime(); err == nil {
		t.Fatal("expected error when FRONTEND_URL not set in production")
	}
}

func TestValidateForRuntime_ProductionRejectsInsecureDB(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Environment: "production",
		FrontendURL: "https://example.com",
		DBSource:    "postgresql://host/db?sslmode=disable",
	}
	if err := cfg.ValidateForRuntime(); err == nil {
		t.Fatal("expected error when DB uses sslmode=disable in production")
	}
}

func TestValidateForRuntime_ProductionAllValidPasses(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Environment:       "production",
		DBSource:          "postgres://user:pass@host:5432/db",
		FrontendURL:       "https://example.com",
		TokenSymmetricKey: "01234567890123456789012345678901",
		S3BucketName:      "my-bucket",
		S3Region:          "ap-southeast-1",
		CloudFrontDomain:  "d123.cloudfront.net",
		GoogleClientID:    "my-client-id.apps.googleusercontent.com",
	}
	if err := cfg.ValidateForRuntime(); err != nil {
		t.Fatalf("unexpected error for valid production config: %v", err)
	}
}
