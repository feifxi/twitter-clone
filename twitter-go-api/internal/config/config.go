package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/spf13/viper"
)

type Config struct {
	Environment              string `mapstructure:"ENVIRONMENT"`
	DBSource                 string `mapstructure:"DATABASE_URL"`
	HTTPServerAddress        string `mapstructure:"HTTP_SERVER_ADDRESS"`
	DBMaxConns               int32  `mapstructure:"DB_MAX_CONNS"`
	DBMinConns               int32  `mapstructure:"DB_MIN_CONNS"`
	DBMaxConnLifetimeMinutes int    `mapstructure:"DB_MAX_CONN_LIFETIME_MINUTES"`
	MaxMediaBytes            int64  `mapstructure:"MAX_MEDIA_BYTES"`
	MaxAvatarBytes           int64  `mapstructure:"MAX_AVATAR_BYTES"`
	MaxBannerBytes           int64  `mapstructure:"MAX_BANNER_BYTES"`
	FrontendURL              string `mapstructure:"FRONTEND_URL"`
	TokenSymmetricKey        string `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	TokenDurationMinutes     int    `mapstructure:"TOKEN_DURATION_MINUTES"`
	RefreshTokenDurationDays int    `mapstructure:"REFRESH_TOKEN_DURATION_DAYS"`
	GoogleClientID           string `mapstructure:"GOOGLE_CLIENT_ID"`
	S3BucketName             string `mapstructure:"S3_BUCKET_NAME"`
	S3Region                 string `mapstructure:"S3_REGION"`
	CloudFrontDomain         string `mapstructure:"CLOUDFRONT_DOMAIN"`
	GatewaySecret            string `mapstructure:"GATEWAY_SECRET"`
	RedisAddress             string `mapstructure:"REDIS_ADDRESS"`
	RedisPassword            string `mapstructure:"REDIS_PASSWORD"`
	SQSEmbeddingQueueURL     string `mapstructure:"SQS_EMBEDDING_QUEUE_URL"`
	GeminiAPIKey             string `mapstructure:"GEMINI_API_KEY"`
	GeminiChatModel          string `mapstructure:"GEMINI_CHAT_MODEL"`
	GeminiEmbeddingModel     string `mapstructure:"GEMINI_EMBEDDING_MODEL"`
	EnableRAG                bool   `mapstructure:"ENABLE_RAG"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("GEMINI_CHAT_MODEL", "gemini-2.5-flash")
	viper.SetDefault("GEMINI_EMBEDDING_MODEL", "gemini-embedding-2-preview")
	viper.SetDefault("DB_MAX_CONNS", int32(25))
	viper.SetDefault("DB_MIN_CONNS", int32(0))
	viper.SetDefault("DB_MAX_CONN_LIFETIME_MINUTES", 5)
	viper.SetDefault("MAX_MEDIA_BYTES", int64(100<<20)) // 100 MiB
	viper.SetDefault("MAX_AVATAR_BYTES", int64(5<<20))  // 5 MiB
	viper.SetDefault("MAX_BANNER_BYTES", int64(10<<20)) // 10 MiB
	// Explicitly bind environment variables so viper.Unmarshal detects them
	// without needing a physical app.env file (which is excluded in CI/CD).
	viper.SetDefault("ENABLE_RAG", true)
	viper.BindEnv("ENVIRONMENT")
	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("HTTP_SERVER_ADDRESS")
	viper.BindEnv("DB_MAX_CONNS")
	viper.BindEnv("DB_MIN_CONNS")
	viper.BindEnv("DB_MAX_CONN_LIFETIME_MINUTES")
	viper.BindEnv("MAX_MEDIA_BYTES")
	viper.BindEnv("MAX_AVATAR_BYTES")
	viper.BindEnv("MAX_BANNER_BYTES")
	viper.BindEnv("FRONTEND_URL")
	viper.BindEnv("TOKEN_SYMMETRIC_KEY")
	viper.BindEnv("TOKEN_DURATION_MINUTES")
	viper.BindEnv("REFRESH_TOKEN_DURATION_DAYS")
	viper.BindEnv("GOOGLE_CLIENT_ID")
	viper.BindEnv("S3_BUCKET_NAME")
	viper.BindEnv("S3_REGION")
	viper.BindEnv("CLOUDFRONT_DOMAIN")
	viper.BindEnv("GATEWAY_SECRET")
	viper.BindEnv("REDIS_ADDRESS")
	viper.BindEnv("REDIS_PASSWORD")
	viper.BindEnv("SQS_EMBEDDING_QUEUE_URL")
	viper.BindEnv("GEMINI_API_KEY")
	viper.BindEnv("GEMINI_CHAT_MODEL")
	viper.BindEnv("GEMINI_EMBEDDING_MODEL")
	viper.BindEnv("ENABLE_RAG")

	viper.AutomaticEnv()

	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		loadFromSSM()
	}

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
		// In production, we MUST have some config from somewhere
		if strings.EqualFold(env, "production") && !configLoadedSuccessfully && len(viper.AllKeys()) == 0 {
			return config, fmt.Errorf("no configuration found: app.env is missing and SSM loading failed")
		}
	}

	if configLoadedSuccessfully {
		fmt.Printf("Metadata: Configuration loaded from AWS SSM (Region: %s, Prefix: %s)\n", os.Getenv("AWS_REGION"), "/chmtwt/prod/")
	}

	err = viper.Unmarshal(&config)
	return
}

var configLoadedSuccessfully bool

func loadFromSSM() {
	prefix := "/chmtwt/prod/"

	region := os.Getenv("AWS_REGION")
	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		fmt.Printf("Error: Unable to initialize AWS SDK: %v (Is AWS_REGION set?)\n", err)
		return
	}

	client := ssm.NewFromConfig(cfg)
	paginator := ssm.NewGetParametersByPathPaginator(client, &ssm.GetParametersByPathInput{
		Path:           &prefix,
		WithDecryption: func() *bool { b := true; return &b }(),
	})

	parameterCount := 0
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Printf("Error: Failed to fetch parameters from SSM at %s: %v\n", prefix, err)
			return
		}
		for _, p := range page.Parameters {
			key := strings.TrimPrefix(*p.Name, prefix)
			val := *p.Value
			if val == "N/A" || val == "none" {
				val = ""
			}
			viper.Set(key, val)
			parameterCount++
		}
	}

	if parameterCount > 0 {
		configLoadedSuccessfully = true
	} else {
		fmt.Printf("⚠️ Warning: No parameters found in SSM path %s\n", prefix)
	}
}

func (c Config) ValidateForRuntime() error {
	if c.Environment != "production" {
		return nil
	}

	if strings.TrimSpace(c.FrontendURL) == "" {
		return fmt.Errorf("FRONTEND_URL is required in production")
	}
	if strings.Contains(strings.ToLower(c.DBSource), "sslmode=disable") {
		return fmt.Errorf("DATABASE_URL must use sslmode in production")
	}
	if strings.TrimSpace(c.GoogleClientID) == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required in production")
	}
	if len(strings.TrimSpace(c.TokenSymmetricKey)) < 32 {
		return fmt.Errorf("TOKEN_SYMMETRIC_KEY must be at least 32 characters in production")
	}
	if strings.TrimSpace(c.S3BucketName) == "" {
		return fmt.Errorf("S3_BUCKET_NAME is required in production")
	}
	if strings.TrimSpace(c.S3Region) == "" {
		return fmt.Errorf("S3_REGION is required in production")
	}
	if strings.TrimSpace(c.CloudFrontDomain) == "" {
		return fmt.Errorf("CLOUDFRONT_DOMAIN is required in production")
	}

	return nil
}

// AllowedOrigins splits the comma-separated FRONTEND_URL into CORS origins.
func (c Config) AllowedOrigins() []string {
	return splitList(c.FrontendURL)
}

func splitList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
