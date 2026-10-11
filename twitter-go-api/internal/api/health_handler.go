package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const readinessTimeout = 2 * time.Second

type HealthResponse struct {
	Status string `json:"status"`
	// Checks maps each dependency to "ok" or "error"; failure details are logged, not exposed.
	Checks map[string]string `json:"checks,omitempty"`
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
// @Description	Checks every dependency (database, and Redis when configured) and reports each one.
// @Tags			Monitoring
// @Produce		json
// @Success		200	{object}	HealthResponse
// @Failure		503	{object}	HealthResponse	"a dependency is unreachable"
// @Router			/readyz [get]
func (s *Server) readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	checks := map[string]func(context.Context) error{"database": s.db.Ping}
	if s.redis != nil {
		checks["redis"] = func(ctx context.Context) error { return s.redis.Ping(ctx).Err() }
	}

	resp := HealthResponse{Status: "ready", Checks: make(map[string]string, len(checks))}
	status := http.StatusOK
	for name, check := range checks {
		if err := check(ctx); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "dependency", name, "err", err)
			resp.Checks[name] = "error"
			resp.Status = "not_ready"
			status = http.StatusServiceUnavailable
			continue
		}
		resp.Checks[name] = "ok"
	}
	c.JSON(status, resp)
}
