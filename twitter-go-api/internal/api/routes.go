package api

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	_ "github.com/chanombude/twitter-go-api/docs" // registers the generated Swagger spec
	"github.com/chanombude/twitter-go-api/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// routes is the API's table of contents: every endpoint, one per line.
//
// Per-route middleware, left to right:
//   - auth:      a valid access token is required (401 otherwise)
//   - optional:  a token is read if present, so responses can be personalized
//   - write:     stricter rate limit for mutations
//   - authLimit: strictest limit, for sign-in endpoints
//   - aiLimit:   limit for calls that cost money (Gemini)
func (s *Server) routes() *gin.Engine {
	r := s.newRouter()

	auth := middleware.AuthMiddleware(s.tokens)
	optional := middleware.OptionalAuthMiddleware(s.tokens)
	write := middleware.RateLimiterWithRedis(s.redis, 2, 5, "rl:write")
	authLimit := middleware.RateLimiterWithRedis(s.redis, 2, 5, "rl:auth")
	aiLimit := middleware.RateLimiterWithRedis(s.redis, 0.2, 5, "rl:ai")

	// Operations (no /api prefix, no rate limit).
	r.GET("/healthz", s.healthz)
	r.GET("/readyz", s.readyz)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	if !s.cfg.Production {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := r.Group("/api/v1", middleware.RateLimiterWithRedis(s.redis, 20, 60, "rl:api"))

	// Auth
	v1.POST("/auth/google", authLimit, s.loginGoogle)
	v1.POST("/auth/refresh", authLimit, s.refreshToken)
	v1.POST("/auth/logout", authLimit, s.logout)
	v1.GET("/auth/me", auth, s.getMe)

	// Users and profiles
	v1.GET("/users/:id", optional, s.getUser)
	v1.GET("/users/by-username/:username", optional, s.getUserByUsername)
	v1.GET("/users/:id/posts", optional, s.getUserPosts)
	v1.GET("/users/:id/replies", optional, s.getUserReplies)
	v1.GET("/users/:id/media", optional, s.getUserMedia)
	v1.GET("/users/:id/likes", auth, s.getUserLikes)
	v1.GET("/users/:id/followers", optional, s.listFollowers)
	v1.GET("/users/:id/following", optional, s.listFollowing)
	v1.PUT("/users/profile", auth, write, s.updateProfile)
	v1.POST("/users/me/onboarding", auth, write, s.completeOnboarding)
	v1.POST("/users/:id/follow", auth, write, s.followUser)
	v1.DELETE("/users/:id/follow", auth, write, s.unfollowUser)

	// Tweets
	v1.GET("/tweets/:id", optional, s.getTweet)
	v1.GET("/tweets/:id/replies", optional, s.getReplies)
	v1.POST("/tweets", auth, write, s.createTweet)
	v1.DELETE("/tweets/:id", auth, write, s.deleteTweet)
	v1.POST("/tweets/:id/like", auth, write, s.likeTweet)
	v1.DELETE("/tweets/:id/like", auth, write, s.unlikeTweet)
	v1.POST("/tweets/:id/retweet", auth, write, s.retweet)
	v1.DELETE("/tweets/:id/retweet", auth, write, s.undoRetweet)
	v1.POST("/uploads/presign", auth, write, s.presignUpload)

	// Timelines
	v1.GET("/feeds/global", optional, s.getGlobalFeed)
	v1.GET("/feeds/following", auth, s.getFollowingFeed)

	// Search and discovery
	v1.GET("/search/users", optional, s.searchUsers)
	v1.GET("/search/tweets", optional, s.searchTweets)
	v1.GET("/search/hashtags", optional, s.searchHashtags)
	v1.GET("/discovery/trending", optional, s.getTrendingHashtags)
	v1.GET("/discovery/users", optional, s.getSuggestedUsers)

	// Notifications
	v1.GET("/notifications", auth, s.listNotifications)
	v1.GET("/notifications/unread-count", auth, s.getUnreadNotificationCount)
	v1.POST("/notifications/mark-read", auth, write, s.markNotificationRead)

	// Direct messages
	v1.GET("/messages/conversations", auth, s.listConversations)
	v1.GET("/messages/conversations/:id/messages", auth, s.listConversationMessages)
	v1.POST("/messages/conversations/:id/messages", auth, write, s.sendMessageToConversation)
	v1.POST("/messages/users/:id/messages", auth, write, s.sendMessageToUser)

	// Real-time (SSE) and assistant
	v1.GET("/events", auth, s.streamEvents)
	v1.POST("/assistant", auth, aiLimit, s.assistant)

	return r
}

// newRouter creates the engine with the middleware every request goes through.
func (s *Server) newRouter() *gin.Engine {
	configureValidationFieldNames()

	r := gin.New()
	// ClientIP (used by rate limiting) only trusts X-Forwarded-For from private
	// ranges, i.e. the reverse proxy in front of the API.
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	r.Use(
		middleware.Metrics(),
		middleware.RequestID(),
		middleware.RequestLogger(),
		gin.Recovery(),
		securityHeaders,
		cors.New(cors.Config{
			AllowOrigins:     s.cfg.AllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length", middleware.RequestIDHeader},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		s.crossOriginProtection,
	)

	// Unknown paths are mostly bot scans: strict limit and no access log.
	r.NoRoute(
		middleware.RateLimiterWithRedis(s.redis, 1, 5, "rl:noroute"),
		func(c *gin.Context) {
			c.Set(middleware.SkipLogKey, true)
			c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Code: "NOT_FOUND", Message: "route not found", RequestID: middleware.GetRequestID(c)})
		},
	)
	return r
}

// crossOriginProtection refuses unsafe requests (POST, PUT, DELETE, …) that a
// browser sends from an origin not in AllowedOrigins. Cookies ride along on
// such requests, so without this any site could act as the signed-in user
// (CSRF). It relies on the Sec-Fetch-Site and Origin headers browsers set;
// requests without them (curl, server-to-server) aren't browser-driven and pass.
func (s *Server) crossOriginProtection(c *gin.Context) {
	if err := s.csrf.Check(c.Request); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Code: "FORBIDDEN", Message: "cross-origin request refused", RequestID: middleware.GetRequestID(c)})
		return
	}
	c.Next()
}

func securityHeaders(c *gin.Context) {
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Header("Cache-Control", "no-store")
	c.Next()
}

// configureValidationFieldNames makes validation errors name fields as clients
// send them (JSON/query/path names) instead of Go struct field names.
func configureValidationFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			if name, _, _ := strings.Cut(field.Tag.Get(tag), ","); name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})
}
