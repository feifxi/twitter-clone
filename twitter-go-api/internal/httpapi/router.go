package httpapi

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	_ "github.com/chanombude/twitter-go-api/docs"
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (s *Server) routes() *gin.Engine {
	configureValidationFieldNames()

	router := gin.New()

	// ClientIP (used by rate limiting) only trusts X-Forwarded-For from private ranges,
	// i.e. a proxy inside the VPC.
	_ = router.SetTrustedProxies([]string{
		"10.0.0.0/8",     // Private IPv4 Class A
		"172.16.0.0/12",  // Private IPv4 Class B
		"192.168.0.0/16", // Private IPv4 Class C
	})

	router.Use(middleware.Metrics())
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger())
	router.Use(gin.Recovery())

	// Handle unmatched routes (silence bot scans and apply strict rate limiting)
	router.NoRoute(
		middleware.RateLimiterWithRedis(s.redis, 1, 5, "rl:noroute"), // Strict: 1 req/sec, burst of 5
		func(c *gin.Context) {
			// Set flag to skip logging this request in logger.GinMiddleware
			c.Set("skip_log", true)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "route not found",
			})
		},
	)

	// Standard security headers.
	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		c.Next()
	})

	router.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/healthz", s.healthz)
	router.GET("/readyz", s.readyz)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if !s.cfg.Production {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := router.Group("/api/v1")
	if s.cfg.GatewaySecret != "" {
		api.Use(middleware.GatewayGuard(s.cfg.GatewaySecret))
	}
	api.Use(middleware.RateLimiterWithRedis(s.redis, 20, 60, "rl:api"))
	s.registerDomainRoutes(api)

	return router
}

func (s *Server) registerDomainRoutes(api *gin.RouterGroup) {
	strictAuthLimiter := middleware.RateLimiterWithRedis(s.redis, 2, 5, "rl:auth")
	strictWriteLimiter := middleware.RateLimiterWithRedis(s.redis, 2, 5, "rl:write")
	optionalAuth := middleware.OptionalAuthMiddleware(s.tokens)
	requiredAuth := middleware.AuthMiddleware(s.tokens)

	s.registerAuthRoutes(api, optionalAuth, requiredAuth, strictAuthLimiter)
	s.registerUserRoutes(api, optionalAuth, requiredAuth, strictWriteLimiter)
	s.registerTweetRoutes(api, optionalAuth, requiredAuth, strictWriteLimiter)
	s.registerFeedRoutes(api, optionalAuth, requiredAuth)
	s.registerSearchRoutes(api, optionalAuth)
	s.registerDiscoveryRoutes(api, optionalAuth)
	s.registerNotificationRoutes(api, requiredAuth)
	s.registerMessageRoutes(api, optionalAuth, requiredAuth, strictWriteLimiter)
	s.registerAssistantRoutes(api, requiredAuth)
}

func (s *Server) registerAuthRoutes(api *gin.RouterGroup, optionalAuth, requiredAuth, strictAuthLimiter gin.HandlerFunc) {
	authPublic := api.Group("/auth")
	authPublic.POST("/google", strictAuthLimiter, s.loginGoogle)
	authPublic.POST("/refresh", strictAuthLimiter, s.refreshToken)
	authPublic.POST("/logout", strictAuthLimiter, optionalAuth, s.logout)

	authPrivate := api.Group("/auth")
	authPrivate.Use(requiredAuth)
	authPrivate.GET("/me", s.getMe)
}

func (s *Server) registerUserRoutes(api *gin.RouterGroup, optionalAuth, requiredAuth, strictWriteLimiter gin.HandlerFunc) {
	usersPublic := api.Group("/users")
	usersPublic.Use(optionalAuth)
	usersPublic.GET("/:id", s.getUser)
	usersPublic.GET("/:id/followers", s.listFollowers)
	usersPublic.GET("/:id/following", s.listFollowing)

	usersPrivate := api.Group("/users")
	usersPrivate.Use(requiredAuth)
	usersPrivate.PUT("/profile", strictWriteLimiter, s.updateProfile)
	usersPrivate.POST("/:id/follow", strictWriteLimiter, s.followUser)
	usersPrivate.DELETE("/:id/follow", s.unfollowUser)
}

func (s *Server) registerTweetRoutes(api *gin.RouterGroup, optionalAuth, requiredAuth, strictWriteLimiter gin.HandlerFunc) {
	tweetsPublic := api.Group("/tweets")
	tweetsPublic.Use(optionalAuth)
	tweetsPublic.GET("/:id", s.getTweet)
	tweetsPublic.GET("/:id/replies", s.getReplies)

	tweetsPrivate := api.Group("/tweets")
	tweetsPrivate.Use(requiredAuth)
	tweetsPrivate.POST("", strictWriteLimiter, s.createTweet)
	tweetsPrivate.DELETE("/:id", s.deleteTweet)
	tweetsPrivate.POST("/:id/like", strictWriteLimiter, s.likeTweet)
	tweetsPrivate.DELETE("/:id/like", s.unlikeTweet)
	tweetsPrivate.POST("/:id/retweet", strictWriteLimiter, s.retweet)
	tweetsPrivate.DELETE("/:id/retweet", s.undoRetweet)

	uploadsPrivate := api.Group("/uploads")
	uploadsPrivate.Use(requiredAuth)
	uploadsPrivate.POST("/presign", strictWriteLimiter, s.presignUpload)
}

func (s *Server) registerFeedRoutes(api *gin.RouterGroup, optionalAuth, requiredAuth gin.HandlerFunc) {
	feedsPublic := api.Group("/feeds")
	feedsPublic.Use(optionalAuth)
	feedsPublic.GET("/global", s.getGlobalFeed)
	feedsPublic.GET("/user/:id", s.getUserFeed)

	feedsPrivate := api.Group("/feeds")
	feedsPrivate.Use(requiredAuth)
	feedsPrivate.GET("/following", s.getFollowingFeed)
}

func (s *Server) registerSearchRoutes(api *gin.RouterGroup, optionalAuth gin.HandlerFunc) {
	searchPublic := api.Group("/search")
	searchPublic.Use(optionalAuth)
	searchPublic.GET("/users", s.searchUsers)
	searchPublic.GET("/tweets", s.searchTweets)
	searchPublic.GET("/hashtags", s.searchHashtags)
}

func (s *Server) registerDiscoveryRoutes(api *gin.RouterGroup, optionalAuth gin.HandlerFunc) {
	discoveryPublic := api.Group("/discovery")
	discoveryPublic.Use(optionalAuth)
	discoveryPublic.GET("/trending", s.getTrendingHashtags)
	discoveryPublic.GET("/users", s.getSuggestedUsers)
}

func (s *Server) registerNotificationRoutes(api *gin.RouterGroup, requiredAuth gin.HandlerFunc) {
	notificationsPrivate := api.Group("/notifications")
	notificationsPrivate.Use(requiredAuth)
	notificationsPrivate.GET("", s.listNotifications)
	notificationsPrivate.GET("/stream", s.streamNotifications)
	notificationsPrivate.GET("/unread-count", s.getUnreadNotificationCount)
	notificationsPrivate.POST("/mark-read", s.markNotificationRead)
}

func (s *Server) registerMessageRoutes(api *gin.RouterGroup, optionalAuth, requiredAuth, strictWriteLimiter gin.HandlerFunc) {
	messagesPublic := api.Group("/messages")
	messagesPublic.GET("/ws", s.streamMessagesWS)

	messagesPrivate := api.Group("/messages")
	messagesPrivate.Use(requiredAuth)
	messagesPrivate.GET("/conversations", s.listConversations)
	messagesPrivate.GET("/conversations/:id/messages", s.listConversationMessages)
	messagesPrivate.POST("/conversations/:id/messages", strictWriteLimiter, s.sendMessageToConversation)
	messagesPrivate.POST("/users/:id/messages", strictWriteLimiter, s.sendMessageToUser)
}

func (s *Server) registerAssistantRoutes(api *gin.RouterGroup, requiredAuth gin.HandlerFunc) {
	assistant := api.Group("/assistant")
	assistant.Use(requiredAuth)
	assistant.POST("", s.assistant)
}

func configureValidationFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			name := parseValidationTagName(field.Tag.Get(tag))
			if name != "" {
				return name
			}
		}
		return field.Name
	})
}

func parseValidationTagName(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return ""
	}
	name := strings.TrimSpace(parts[0])
	if name == "" || name == "-" {
		return ""
	}
	return name
}
