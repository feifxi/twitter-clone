package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const readinessTimeout = 2 * time.Second

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service,omitempty"`
}

// healthz godoc
// @Summary		Liveness Probe
// @Description	Reports that the process is up. Never checks dependencies, so a database blip doesn't get the container restarted.
// @Tags			Monitoring
// @Produce		json
// @Success		200	{object}	HealthResponse
// @Router			/healthz [get]
func (s *Server) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// readyz godoc
// @Summary		Readiness Probe
// @Description	Verifies the database (and Redis, when configured) are reachable.
// @Tags			Monitoring
// @Produce		json
// @Success		200	{object}	HealthResponse
// @Failure		503	{object}	HealthResponse	"dependency failure"
// @Router			/readyz [get]
func (s *Server) readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "not_ready", Service: "database"})
		return
	}
	if s.redis != nil {
		if err := s.redis.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "not_ready", Service: "redis"})
			return
		}
	}
	c.JSON(http.StatusOK, HealthResponse{Status: "ready"})
}
