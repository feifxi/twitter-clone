// Package api exposes the feature services over HTTP (REST and SSE)
// with Gin. Handlers only bind input, call a service and map the result to a DTO.
package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/chanombude/twitter-go-api/internal/realtime"
	"github.com/chanombude/twitter-go-api/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Production bool
	// AllowedOrigins may call the API from a browser with credentials (CORS) and
	// send unsafe requests (cross-origin protection).
	AllowedOrigins []string
	// CookieDomain scopes the session cookies; empty means the API host only.
	CookieDomain string
}

// pinger is a dependency checked by the readiness probe.
type pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	cfg    Config
	svc    Services
	tokens token.Maker
	hub    *realtime.Hub
	db     pinger
	redis  *redis.Client // optional: shared rate limits and readiness
	csrf   *http.CrossOriginProtection
	router *gin.Engine
}

func New(cfg Config, svc Services, tokens token.Maker, hub *realtime.Hub, db pinger, rdb *redis.Client) (*Server, error) {
	csrf := http.NewCrossOriginProtection()
	for _, origin := range cfg.AllowedOrigins {
		if err := csrf.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("allowed origin: %w", err)
		}
	}

	s := &Server{
		cfg:    cfg,
		svc:    svc,
		tokens: tokens,
		hub:    hub,
		db:     db,
		redis:  rdb,
		csrf:   csrf,
	}
	s.router = s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

// HTTPServer returns a server with conservative timeouts. WriteTimeout is
// disabled because SSE and assistant streams stay open longer than any fixed
// limit; instead every request context derives from baseCtx, so cancelling it
// on shutdown ends open streams rather than waiting them out.
func (s *Server) HTTPServer(baseCtx context.Context, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s.router,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

type idURIRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}
