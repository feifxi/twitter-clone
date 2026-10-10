package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validKey = "01234567890123456789012345678901"

func TestLoad_DefaultsAndTypedValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("TOKEN_SYMMETRIC_KEY", validKey)
	t.Setenv("ACCESS_TOKEN_TTL", "30m")
	t.Setenv("ALLOWED_ORIGINS", "https://a.com , ,https://b.com")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AccessTokenTTL != 30*time.Minute || cfg.RefreshTokenTTL != 720*time.Hour {
		t.Fatalf("unexpected TTLs: %v %v", cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	}
	if got := strings.Join(cfg.AllowedOrigins, "|"); got != "https://a.com|https://b.com" {
		t.Fatalf("unexpected origins: %q", got)
	}
	if cfg.LogLevel != slog.LevelDebug || cfg.Production() {
		t.Fatalf("unexpected level/env: %v %q", cfg.LogLevel, cfg.Environment)
	}
}

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("TOKEN_SYMMETRIC_KEY", validKey)

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestValidate(t *testing.T) {
	prod := Config{
		Environment:       "production",
		DatabaseURL:       "postgres://user:pass@host:5432/db?sslmode=require",
		AllowedOrigins:    []string{"https://twitter.example.com"},
		TokenSymmetricKey: validKey,
		S3Bucket:          "bucket",
		CloudFrontDomain:  "d123.cloudfront.net",
		GoogleClientID:    "client-id",
	}
	if err := prod.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}

	cases := map[string]func(c *Config){
		"short token key":        func(c *Config) { c.TokenSymmetricKey = "short" },
		"origin with a path":     func(c *Config) { c.AllowedOrigins = []string{"https://twitter.example.com/app"} },
		"origin without scheme":  func(c *Config) { c.AllowedOrigins = []string{"twitter.example.com"} },
		"insecure database":      func(c *Config) { c.DatabaseURL += "&sslmode=disable" },
		"missing bucket":         func(c *Config) { c.S3Bucket = "" },
		"localhost origin":       func(c *Config) { c.AllowedOrigins = append(c.AllowedOrigins, "http://localhost:3000") },
		"missing google client":  func(c *Config) { c.GoogleClientID = " " },
		"missing cloudfront cdn": func(c *Config) { c.CloudFrontDomain = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := prod
			c.AllowedOrigins = append([]string(nil), prod.AllowedOrigins...)
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	dev := Config{Environment: "development", TokenSymmetricKey: validKey, DatabaseURL: "postgres://localhost/db?sslmode=disable"}
	if err := dev.Validate(); err != nil {
		t.Fatalf("development config should not require production settings: %v", err)
	}
}

// The documented example must stay loadable and match the config's variable names.
func TestExampleEnvFileLoads(t *testing.T) {
	example, err := os.ReadFile("../../app.env.example")
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.env"), example, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, name := range []string{"DATABASE_URL", "TOKEN_SYMMETRIC_KEY", "CLOUDFRONT_DOMAIN", "S3_BUCKET_NAME", "ENVIRONMENT"} {
		t.Setenv(name, "") // let app.env provide them
		os.Unsetenv(name)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("example env file does not load: %v", err)
	}
	if cfg.CloudFrontDomain != "" || cfg.Environment != "development" || cfg.MaxBannerBytes != 10<<20 {
		t.Fatalf("inline comments were not stripped: %+v", cfg)
	}
}
