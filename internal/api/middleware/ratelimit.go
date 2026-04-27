package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

type RateLimiter struct {
	redis  *redis.Client
	limits map[string]RateLimitConfig
}

func NewRateLimiter(redis *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redis,
		limits: map[string]RateLimitConfig{
			"default": {
				MaxRequests: 100,
				Window:      time.Minute,
			},
			"auth": {
				MaxRequests: 10,
				Window:      time.Minute,
			},
			"admin": {
				MaxRequests: 30,
				Window:      time.Minute,
			},
		},
	}
}

func (rl *RateLimiter) Limit(identifier string, config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "middleware.RateLimiter.Limit"

		ctx := c.Request.Context()
		key := fmt.Sprintf("ratelimit:%s", identifier)

		allowed, remaining, resetTime, err := rl.checkLimit(ctx, key, config)
		if err != nil {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limit_exceeded",
				"error_description": "Too many requests. Please try again later.",
				"retry_after":       int(time.Until(resetTime).Seconds()) + 1,
			})
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) checkLimit(ctx context.Context, key string, config RateLimitConfig) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Truncate(config.Window)
	resetTime := windowStart.Add(config.Window)

	currentWindowKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())
	previousWindowKey := fmt.Sprintf("%s:%d", key, windowStart.Add(-config.Window).Unix())

	var prevCount int64
	pipe := rl.redis.Pipeline()
	pipe.Get(ctx, previousWindowKey)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
	}

	prevCountStr, _ := rl.redis.Get(ctx, previousWindowKey).Result()
	if prevCountStr != "" {
		fmt.Sscanf(prevCountStr, "%d", &prevCount)
	}

	weight := float64(time.Since(windowStart)) / float64(config.Window)
	prevWeightedCount := float64(prevCount) * (1 - weight)

	pipe = rl.redis.Pipeline()
	incr := pipe.Incr(ctx, currentWindowKey)
	pipe.Expire(ctx, currentWindowKey, config.Window*2)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return true, config.MaxRequests, resetTime, fmt.Errorf("redis pipeline: %w", err)
	}

	currentCount := int(incr.Val())
	totalCount := int(float64(currentCount) + prevWeightedCount)

	remaining := config.MaxRequests - totalCount
	if remaining < 0 {
		remaining = 0
	}

	allowed := totalCount <= config.MaxRequests

	return allowed, remaining, resetTime, nil
}

func (rl *RateLimiter) IPLimit() gin.HandlerFunc {
	return rl.Limit("ip:default", rl.limits["default"])
}

func (rl *RateLimiter) AuthLimit() gin.HandlerFunc {
	return rl.Limit("ip:auth", rl.limits["auth"])
}

func (rl *RateLimiter) AdminLimit() gin.HandlerFunc {
	return rl.Limit("ip:admin", rl.limits["admin"])
}