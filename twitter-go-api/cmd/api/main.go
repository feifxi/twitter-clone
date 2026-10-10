package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/chanombude/twitter-go-api/docs"
	"github.com/chanombude/twitter-go-api/internal/assistant"
	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/discovery"
	"github.com/chanombude/twitter-go-api/internal/embedding"
	"github.com/chanombude/twitter-go-api/internal/feed"
	"github.com/chanombude/twitter-go-api/internal/httpapi"
	"github.com/chanombude/twitter-go-api/internal/logger"
	"github.com/chanombude/twitter-go-api/internal/media"
	"github.com/chanombude/twitter-go-api/internal/message"
	"github.com/chanombude/twitter-go-api/internal/notification"
	"github.com/chanombude/twitter-go-api/internal/realtime"
	"github.com/chanombude/twitter-go-api/internal/search"
	"github.com/chanombude/twitter-go-api/internal/storage"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/chanombude/twitter-go-api/internal/tweet"
	"github.com/chanombude/twitter-go-api/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// @title			Twitter Go API
// @version		1.0
// @description	High-performance Go backend for the Chanom Twitter clone.
// @termsOfService	http://swagger.io/terms/

// @contact.name	API Support
// @contact.url	http://github.com/chanombude/twitter-go-api

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host			localhost:8080
// @BasePath		/api/v1

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type 'Bearer ' followed by your JWT token.

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("API exited with error")
	}
}

func run() error {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.ValidateForRuntime(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	logger.InitLogger(cfg.Environment)
	production := strings.EqualFold(cfg.Environment, "production")
	if production {
		gin.SetMode(gin.ReleaseMode)
	}

	// Cancelled on SIGINT/SIGTERM; every long-lived goroutine and request derives from it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := newDBPool(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(cfg.DBSource); err != nil {
		return err
	}
	log.Info().Msg("database migrated")

	rdb := newRedis(ctx, cfg)
	if rdb != nil {
		defer rdb.Close()
	}

	srv, err := newServer(ctx, cfg, db.NewStore(pool), rdb, production)
	if err != nil {
		return err
	}

	httpServer := srv.HTTPServer(ctx, cfg.HTTPServerAddress)
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.ListenAndServe() }()
	log.Info().Str("addr", cfg.HTTPServerAddress).Msg("API listening")

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
	case <-ctx.Done():
		log.Info().Msg("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// newServer wires the feature services into the HTTP layer.
func newServer(ctx context.Context, cfg config.Config, store db.Store, rdb *redis.Client, production bool) (*httpapi.Server, error) {
	tokens, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}

	region := cfg.S3Region
	if region == "" {
		region = "ap-southeast-1"
	}

	s3, err := storage.NewS3(ctx, storage.Config{Bucket: cfg.S3BucketName, Region: region, CDNDomain: cfg.CloudFrontDomain})
	if err != nil {
		return nil, fmt.Errorf("create storage: %w", err)
	}

	queueURL := cfg.SQSEmbeddingQueueURL
	if !cfg.EnableRAG {
		queueURL = ""
	}
	embeddings, err := embedding.NewPublisher(ctx, queueURL, region)
	if err != nil {
		return nil, fmt.Errorf("create embedding publisher: %w", err)
	}

	gemini, err := assistant.NewGeminiClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	hub := realtime.NewHub(rdb)
	go hub.Run(ctx)
	notifications := notification.New(store, hub)

	services := httpapi.Services{
		Auth: auth.New(auth.Config{
			GoogleClientID:  cfg.GoogleClientID,
			AccessTokenTTL:  time.Duration(cfg.TokenDurationMinutes) * time.Minute,
			RefreshTokenTTL: time.Duration(cfg.RefreshTokenDurationDays) * 24 * time.Hour,
		}, store, tokens, auth.GoogleIDTokenVerifier{}),
		Users:         user.New(store, s3, notifications),
		Tweets:        tweet.New(store, s3, embeddings, notifications),
		Feeds:         feed.New(store),
		Search:        search.New(store),
		Discovery:     discovery.New(store),
		Notifications: notifications,
		Messages:      message.New(store),
		Assistant: assistant.New(assistant.Config{
			ChatModel:      cfg.GeminiChatModel,
			EmbeddingModel: cfg.GeminiEmbeddingModel,
			EnableRAG:      cfg.EnableRAG,
		}, store, gemini),
		Media: media.New(s3, media.Limits{
			MaxTweetMediaBytes: cfg.MaxMediaBytes,
			MaxAvatarBytes:     cfg.MaxAvatarBytes,
			MaxBannerBytes:     cfg.MaxBannerBytes,
		}),
	}

	return httpapi.New(httpapi.Config{
		Production:     production,
		AllowedOrigins: cfg.AllowedOrigins(),
		GatewaySecret:  cfg.GatewaySecret,
	}, services, tokens, hub, store, rdb), nil
}

func newDBPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DBSource)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	if cfg.DBMaxConns > 0 {
		poolCfg.MaxConns = cfg.DBMaxConns
	}
	if cfg.DBMinConns > 0 {
		poolCfg.MinConns = cfg.DBMinConns
	}
	if cfg.DBMaxConnLifetimeMinutes > 0 {
		poolCfg.MaxConnLifetime = time.Duration(cfg.DBMaxConnLifetimeMinutes) * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return pool, nil
}

// newRedis returns nil when Redis is not configured or unreachable; the API then
// runs single-instance (in-memory rate limits and realtime fan-out).
func newRedis(ctx context.Context, cfg config.Config) *redis.Client {
	if cfg.RedisAddress == "" {
		return nil
	}

	opts, err := redis.ParseURL(cfg.RedisAddress)
	if err != nil {
		log.Warn().Err(err).Msg("invalid REDIS_ADDRESS, starting without Redis")
		return nil
	}
	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}

	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis unavailable, starting without Redis")
		_ = client.Close()
		return nil
	}
	return client
}
