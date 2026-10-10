package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/chanombude/twitter-go-api/docs"
	"github.com/chanombude/twitter-go-api/internal/api"
	"github.com/chanombude/twitter-go-api/internal/assistant"
	"github.com/chanombude/twitter-go-api/internal/auth"
	"github.com/chanombude/twitter-go-api/internal/config"
	"github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/discovery"
	"github.com/chanombude/twitter-go-api/internal/embedding"
	"github.com/chanombude/twitter-go-api/internal/feed"
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
		slog.Error("API exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.SetDefault(logger.New(os.Stderr, cfg.Production(), cfg.LogLevel))
	if cfg.Production() {
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

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	slog.Info("database migrated")

	rdb := newRedis(ctx, cfg)
	if rdb != nil {
		defer rdb.Close()
	}

	srv, err := newServer(ctx, cfg, db.NewStore(pool), rdb)
	if err != nil {
		return err
	}

	httpServer := srv.HTTPServer(ctx, cfg.HTTPAddr)
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.ListenAndServe() }()
	slog.Info("API listening", "addr", cfg.HTTPAddr)

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// newServer wires the feature services into the HTTP layer.
func newServer(ctx context.Context, cfg config.Config, store db.Store, rdb *redis.Client) (*api.Server, error) {
	tokens, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}

	s3, err := storage.NewS3(ctx, storage.Config{Bucket: cfg.S3Bucket, Region: cfg.S3Region, CDNDomain: cfg.CloudFrontDomain})
	if err != nil {
		return nil, fmt.Errorf("create storage: %w", err)
	}

	gemini, err := assistant.NewGeminiClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	// RAG needs Gemini both to embed tweets (background worker) and to embed questions.
	rag := cfg.EnableRAG && gemini != nil
	if rag {
		docs := embedding.NewGemini(gemini, cfg.GeminiEmbeddingModel, embedding.TaskDocument)
		go embedding.NewWorker(store, docs, embedding.DefaultWorkerConfig()).Run(ctx)
	} else {
		slog.Warn("RAG disabled: tweets are not embedded (set ENABLE_RAG and GEMINI_API_KEY to enable)")
	}

	hub := realtime.NewHub(rdb)
	go hub.Run(ctx)
	notifications := notification.New(store, hub)

	services := api.Services{
		Auth: auth.New(auth.Config{
			GoogleClientID:  cfg.GoogleClientID,
			AccessTokenTTL:  cfg.AccessTokenTTL,
			RefreshTokenTTL: cfg.RefreshTokenTTL,
		}, store, tokens, auth.GoogleIDTokenVerifier{}),
		Users:         user.New(store, s3, notifications),
		Tweets:        tweet.New(tweet.Config{EmbedTweets: rag}, store, s3, notifications),
		Feeds:         feed.New(store),
		Search:        search.New(store),
		Discovery:     discovery.New(store),
		Notifications: notifications,
		Messages:      message.New(store, hub),
		Assistant: assistant.New(assistant.Config{
			ChatModel: cfg.GeminiChatModel,
			EnableRAG: rag,
		}, store, gemini, embedding.NewGemini(gemini, cfg.GeminiEmbeddingModel, embedding.TaskQuery)),
		Media: media.New(s3, media.Limits{
			MaxTweetMediaBytes: cfg.MaxMediaBytes,
			MaxAvatarBytes:     cfg.MaxAvatarBytes,
			MaxBannerBytes:     cfg.MaxBannerBytes,
		}),
	}

	return api.New(api.Config{
		Production:     cfg.Production(),
		AllowedOrigins: cfg.AllowedOrigins,
		GatewaySecret:  cfg.GatewaySecret,
	}, services, tokens, hub, store, rdb), nil
}

func newDBPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	if cfg.DBMaxConns > 0 {
		poolCfg.MaxConns = cfg.DBMaxConns
	}
	if cfg.DBMinConns > 0 {
		poolCfg.MinConns = cfg.DBMinConns
	}
	if cfg.DBMaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
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
	if cfg.RedisURL == "" {
		return nil
	}

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Warn("invalid REDIS_URL, starting without Redis", "err", err)
		return nil
	}

	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		slog.Warn("Redis unavailable, starting without Redis", "err", err)
		_ = client.Close()
		return nil
	}
	return client
}
