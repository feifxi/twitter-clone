package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chanombude/twitter-go-api/internal/config"
	db "github.com/chanombude/twitter-go-api/internal/db"
	"github.com/chanombude/twitter-go-api/internal/service"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/chanombude/twitter-go-api/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Server struct {
	config      config.Config
	store       db.Store
	tokenMaker  token.Maker
	storage     service.StorageService
	authUC      usecase.AuthService
	userUC      usecase.UserService
	tweetUC     usecase.TweetService
	feedUC      usecase.FeedService
	searchUC    usecase.SearchService
	discoveryUC usecase.DiscoveryService
	notifyUC    usecase.NotificationService
	messageUC   usecase.MessageService
	assistantUC usecase.AssistantService
	uploadUC    usecase.UploadService
	router      *gin.Engine
	redis       *redis.Client
	sseClients  map[int64][]*sseClient
	sseMu       sync.RWMutex
	wsClients   map[int64]map[*chatWSClient]struct{}
	wsMu        sync.RWMutex
	done        chan struct{}
}

type idURIRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func NewServer(config config.Config, store db.Store, redisClient *redis.Client) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	storageService, err := service.NewS3StorageService(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage service: %w", err)
	}

	embeddingPublisher, err := service.NewSQSEmbeddingPublisher(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create sqs embedding publisher: %w", err)
	}

	genaiClient, err := usecase.NewGeminiClient(context.Background(), config.GeminiAPIKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create gemini client: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		storage:    storageService,
		redis:      redisClient,
		sseClients: make(map[int64][]*sseClient),
		wsClients:  make(map[int64]map[*chatWSClient]struct{}),
		done:       make(chan struct{}),
	}
	server.authUC = usecase.NewAuthUsecase(config, store, tokenMaker, usecase.NewRealGoogleVerifier())
	server.userUC = usecase.NewUserUsecase(store, storageService, server.publishNotification)
	server.tweetUC = usecase.NewTweetUsecase(config, store, storageService, embeddingPublisher, server.publishNotification)
	server.feedUC = usecase.NewFeedUsecase(store)
	server.searchUC = usecase.NewSearchUsecase(store)
	server.discoveryUC = usecase.NewDiscoveryUsecase(store)
	server.notifyUC = usecase.NewNotificationUsecase(store)
	server.messageUC = usecase.NewMessageUsecase(store)
	server.assistantUC = usecase.NewAssistantUsecase(config, store, genaiClient)
	server.uploadUC = usecase.NewUploadUsecase(config, storageService)

	server.setupRouter()

	if redisClient != nil {
		go server.listenRedisNotifications()
	}

	return server, nil
}

// Shutdown signals background goroutines (e.g. Redis listener) to stop.
func (server *Server) Shutdown() {
	close(server.done)
}

func (server *Server) HTTPServer(address string) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           server.router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		// Keep write timeout disabled to support long-lived SSE streams.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
}

type redisNotificationPayload struct {
	RecipientID  int64                `json:"recipientId"`
	Notification NotificationResponse `json:"notification"`
}

func (server *Server) publishNotification(notification db.Notification) {
	hydrateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hydrated, err := server.notifyUC.HydrateNotification(hydrateCtx, notification)
	if err != nil {
		log.Error().Err(err).Int64("notification_id", notification.ID).Int64("recipient_id", notification.RecipientID).Msg("Failed to hydrate notification for SSE; event not published")
		return
	}
	server.broadcastToRedis(notification.RecipientID, newNotificationResponse(hydrated))
}

func (server *Server) broadcastToRedis(recipientID int64, notification NotificationResponse) {
	if server.redis == nil {
		server.sendNotificationToUser(recipientID, notification)
		return
	}

	payload := redisNotificationPayload{RecipientID: recipientID, Notification: notification}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal redis notification payload")
		return
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.redis.Publish(pubCtx, "notifications", data).Err(); err != nil {
		log.Error().Err(err).Msg("Failed to publish notification to Redis")
		server.sendNotificationToUser(recipientID, notification)
	}
}
