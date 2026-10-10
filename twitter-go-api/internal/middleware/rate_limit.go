package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides in-memory token-bucket limiting (single-instance).
func RateLimiter(r rate.Limit, b int) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(r, b)}
		}
		clients[ip].lastSeen = time.Now()
		limiter := clients[ip].limiter
		mu.Unlock()

		if !limiter.Allow() {
			abortWithError(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "rate limit exceeded")
			return
		}

		c.Next()
	}
}

var redisWindowScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("TTL", KEYS[1])
return {current, ttl}
`)

// RateLimiterWithRedis uses Redis for cross-instance limiting and falls back to local memory when redis is nil or unavailable.
func RateLimiterWithRedis(redisClient *redis.Client, r rate.Limit, b int, prefix string) gin.HandlerFunc {
	localFallback := RateLimiter(r, b)
	if redisClient == nil || r <= 0 || b <= 0 {
		return localFallback
	}

	windowSeconds := int(math.Ceil(float64(b) / float64(r)))
	if windowSeconds < 1 {
		windowSeconds = 1
	}

	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		key := fmt.Sprintf("%s:%s:%s", prefix, path, c.ClientIP())

		redisCtx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		raw, err := redisWindowScript.Run(redisCtx, redisClient, []string{key}, windowSeconds).Result()
		if err != nil {
			localFallback(c)
			return
		}

		results, ok := raw.([]any)
		if !ok || len(results) != 2 {
			localFallback(c)
			return
		}

		current, okCurrent := toInt64(results[0])
		ttl, okTTL := toInt64(results[1])
		if !okCurrent || !okTTL {
			localFallback(c)
			return
		}

		limit := int64(b)
		c.Header("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		remaining := limit - current
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if current > limit {
			retryAfter := max(ttl, 1)
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			abortWithError(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "rate limit exceeded")
			return
		}

		c.Next()
	}
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case uint64:
		return int64(n), true
	case string:
		p, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, false
		}
		return p, true
	default:
		return 0, false
	}
}
