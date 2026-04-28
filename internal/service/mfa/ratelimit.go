package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	redis *redis.Client
}

func NewRedisRateLimiter(redisClient *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{redis: redisClient}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	const op = "RedisRateLimiter.Allow"

	counterKey := fmt.Sprintf("rate_limit:%s", key)

	count, err := r.redis.Incr(ctx, counterKey).Result()
	if err != nil {
		return false, fmt.Errorf("%s: incr: %w", op, err)
	}

	if count == 1 {
		r.redis.Expire(ctx, counterKey, window)
	}

	return count <= int64(limit), nil
}