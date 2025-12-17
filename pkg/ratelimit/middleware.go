package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"yeti/pkg/metrics"
)

type Limiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	mu       sync.Mutex
}

type RateLimitConfig struct {
	RPS             float64
	Burst           int
	CleanupInterval time.Duration
	MaxAge          time.Duration
}

func DefaultConfig() RateLimitConfig {
	return RateLimitConfig{
		RPS:             10.0,
		Burst:           20,
		CleanupInterval: 5 * time.Minute,
		MaxAge:          10 * time.Minute,
	}
}

func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	limiters := make(map[string]*Limiter)
	var mu sync.RWMutex

	go func() {
		ticker := time.NewTicker(config.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, limiter := range limiters {
				limiter.mu.Lock()
				lastSeen := limiter.lastSeen
				limiter.mu.Unlock()
				if now.Sub(lastSeen) > config.MaxAge {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = c.RemoteIP()
		}

		mu.RLock()
		limiter, exists := limiters[clientIP]
		mu.RUnlock()

		if !exists {
			mu.Lock()
			limiter, exists = limiters[clientIP]
			if !exists {
				limiter = &Limiter{
					limiter:  rate.NewLimiter(rate.Limit(config.RPS), config.Burst),
					lastSeen: time.Now(),
				}
				limiters[clientIP] = limiter
			}
			mu.Unlock()
		}

		limiter.mu.Lock()
		limiter.lastSeen = time.Now()
		limiter.mu.Unlock()

		if !limiter.limiter.Allow() {
			metrics.RateLimitRequestsTotal.WithLabelValues("limited").Inc()
			c.Header("X-RateLimit-Limit", formatRate(config.RPS))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "1")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "rate limit exceeded",
				"error_code": "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		metrics.RateLimitRequestsTotal.WithLabelValues("allowed").Inc()

		c.Header("X-RateLimit-Limit", formatRate(config.RPS))
		remaining := limiter.limiter.Burst() - int(limiter.limiter.Tokens())
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		c.Next()
	}
}

func formatRate(rps float64) string {
	return strconv.Itoa(int(rps))
}
