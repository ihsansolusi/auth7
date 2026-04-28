package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Threshold struct {
	Failures int
	Cooldown time.Duration
	Lockout  bool
}

type BruteForceProtector struct {
	redis      *redis.Client
	thresholds []Threshold
}

func NewBruteForceProtector(redis *redis.Client) *BruteForceProtector {
	return &BruteForceProtector{
		redis: redis,
		thresholds: []Threshold{
			{Failures: 4, Cooldown: 1 * time.Minute, Lockout: false},
			{Failures: 6, Cooldown: 5 * time.Minute, Lockout: false},
			{Failures: 10, Cooldown: 0, Lockout: true},
		},
	}
}

func (bf *BruteForceProtector) RecordFailure(ctx context.Context, username, ip string) error {
	const op = "BruteForceProtector.RecordFailure"

	userKey := fmt.Sprintf("bruteforce:user:%s", username)
	ipKey := fmt.Sprintf("bruteforce:ip:%s", ip)

	pipe := bf.redis.Pipeline()

	pipe.Incr(ctx, userKey)
	pipe.Expire(ctx, userKey, 1*time.Hour)

	pipe.Incr(ctx, ipKey)
	pipe.Expire(ctx, ipKey, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: redis pipeline: %w", op, err)
	}

	return nil
}

func (bf *BruteForceProtector) RecordSuccess(ctx context.Context, username, ip string) error {
	const op = "BruteForceProtector.RecordSuccess"

	userKey := fmt.Sprintf("bruteforce:user:%s", username)
	ipKey := fmt.Sprintf("bruteforce:ip:%s", ip)

	pipe := bf.redis.Pipeline()

	pipe.Del(ctx, userKey)
	pipe.Del(ctx, ipKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: redis pipeline: %w", op, err)
	}

	return nil
}

func (bf *BruteForceProtector) IsLocked(ctx context.Context, username, ip string) (bool, time.Duration, error) {
	const op = "BruteForceProtector.IsLocked"

	userKey := fmt.Sprintf("bruteforce:user:%s", username)
	ipKey := fmt.Sprintf("bruteforce:ip:%s", ip)

	userCountStr, err := bf.redis.Get(ctx, userKey).Result()
	if err != nil && err != redis.Nil {
		return false, 0, fmt.Errorf("%s: redis get user: %w", op, err)
	}

	ipCountStr, err := bf.redis.Get(ctx, ipKey).Result()
	if err != nil && err != redis.Nil {
		return false, 0, fmt.Errorf("%s: redis get ip: %w", op, err)
	}

	var userCount, ipCount int
	if userCountStr != "" {
		userCount, _ = strconv.Atoi(userCountStr)
	}
	if ipCountStr != "" {
		ipCount, _ = strconv.Atoi(ipCountStr)
	}

	maxCount := userCount
	if ipCount > maxCount {
		maxCount = ipCount
	}

	for _, threshold := range bf.thresholds {
		if maxCount >= threshold.Failures {
			if threshold.Lockout {
				ttl, _ := bf.redis.TTL(ctx, userKey).Result()
				return true, ttl, nil
			}

			cooldownKey := fmt.Sprintf("bruteforce:cooldown:%s", username)
			ttl, _ := bf.redis.TTL(ctx, cooldownKey).Result()

			if ttl > 0 {
				return true, ttl, nil
			}

			userTtl, _ := bf.redis.TTL(ctx, userKey).Result()
			return true, userTtl, nil
		}
	}

	return false, 0, nil
}

func (bf *BruteForceProtector) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "BruteForceProtector.Middleware"

		if c.FullPath() != "/auth/login" && c.FullPath() != "/oauth2/token" {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		username := c.PostForm("username")
		clientID := c.PostForm("client_id")

		identifier := username
		if identifier == "" {
			identifier = clientID
		}
		if identifier == "" {
			identifier = c.ClientIP()
		}

		locked, retryAfter, err := bf.IsLocked(ctx, identifier, c.ClientIP())
		if err != nil {
			c.Next()
			return
		}

		if locked {
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":             "account_locked",
				"error_description": "Too many failed attempts. Please try again later.",
				"retry_after":       int(retryAfter.Seconds()) + 1,
			})
			return
		}

		c.Set("bruteforce_username", identifier)
		c.Next()
	}
}

func (bf *BruteForceProtector) RecordLoginResult(ctx context.Context, username, ip string, success bool) {
	if success {
		bf.RecordSuccess(ctx, username, ip)
	} else {
		bf.RecordFailure(ctx, username, ip)
	}
}