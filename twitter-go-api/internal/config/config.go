// Package config loads runtime settings from environment variables. Values are
// injected by the environment (Docker, systemd, CI); app.env is only a local-dev
// convenience and never overrides variables that are already set.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const minTokenKeyLength = 32

type Config struct {
	Environment string     `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    slog.Level `env:"LOG_LEVEL" envDefault:"info"`
	HTTPAddr    string     `env:"HTTP_SERVER_ADDRESS" envDefault:"0.0.0.0:8080"`

	DatabaseURL       string        `env:"DATABASE_URL,required,notEmpty"`
	DBMaxConns        int32         `env:"DB_MAX_CONNS" envDefault:"25"`
	DBMinConns        int32         `env:"DB_MIN_CONNS" envDefault:"0"`
	DBMaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"5m"`

	// RedisURL is optional; without it the API runs single-instance (in-memory rate limits and realtime).
	RedisURL string `env:"REDIS_URL"`

	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`

	TokenSymmetricKey string        `env:"TOKEN_SYMMETRIC_KEY,required,notEmpty"`
	AccessTokenTTL    time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL   time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	GoogleClientID    string        `env:"GOOGLE_CLIENT_ID"`

	S3Bucket         string `env:"S3_BUCKET_NAME"`
	S3Region         string `env:"S3_REGION" envDefault:"ap-southeast-1"`
	CloudFrontDomain string `env:"CLOUDFRONT_DOMAIN"`
	MaxMediaBytes    int64  `env:"MAX_MEDIA_BYTES" envDefault:"104857600"` // 100 MiB
	MaxAvatarBytes   int64  `env:"MAX_AVATAR_BYTES" envDefault:"5242880"`  // 5 MiB
	MaxBannerBytes   int64  `env:"MAX_BANNER_BYTES" envDefault:"10485760"` // 10 MiB

	GeminiAPIKey         string `env:"GEMINI_API_KEY"`
	GeminiChatModel      string `env:"GEMINI_CHAT_MODEL" envDefault:"gemini-2.5-flash"`
	GeminiEmbeddingModel string `env:"GEMINI_EMBEDDING_MODEL" envDefault:"gemini-embedding-2-preview"`
	EnableRAG            bool   `env:"ENABLE_RAG" envDefault:"true"`
}

// Load reads app.env (if present) and the environment, then validates the result.
func Load() (Config, error) {
	if err := godotenv.Load("app.env"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("read app.env: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	cfg.AllowedOrigins = trimList(cfg.AllowedOrigins)
	return cfg, cfg.Validate()
}

// trimList drops blanks and surrounding spaces ("a, ,b" → [a b]).
func trimList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (c Config) Production() bool {
	return strings.EqualFold(c.Environment, "production")
}

// Validate rejects settings that are unsafe everywhere, plus anything production can't run without.
func (c Config) Validate() error {
	var errs []error
	if len(c.TokenSymmetricKey) < minTokenKeyLength {
		errs = append(errs, fmt.Errorf("TOKEN_SYMMETRIC_KEY must be at least %d characters", minTokenKeyLength))
	}

	if c.Production() {
		if strings.Contains(strings.ToLower(c.DatabaseURL), "sslmode=disable") {
			errs = append(errs, errors.New("DATABASE_URL must not disable TLS in production"))
		}
		for _, field := range []struct{ name, value string }{
			{"GOOGLE_CLIENT_ID", c.GoogleClientID},
			{"S3_BUCKET_NAME", c.S3Bucket},
			{"CLOUDFRONT_DOMAIN", c.CloudFrontDomain},
		} {
			if strings.TrimSpace(field.value) == "" {
				errs = append(errs, fmt.Errorf("%s is required in production", field.name))
			}
		}
		for _, origin := range c.AllowedOrigins {
			if strings.Contains(origin, "localhost") {
				errs = append(errs, fmt.Errorf("ALLOWED_ORIGINS must not include %q in production", origin))
			}
		}
	}
	return errors.Join(errs...)
}
