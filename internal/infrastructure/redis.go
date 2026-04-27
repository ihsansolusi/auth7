package infrastructure

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/ihsansolusi/auth7/pkg/config"
)

func NewRedis(ctx context.Context, cfg config.RedisConfig, logger zerolog.Logger) (*redis.Client, error) {
	const op = "infrastructure.NewRedis"

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.DSN,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	logger.Info().
		Str("op", op).
		Int("pool_size", cfg.PoolSize).
		Msg("redis client initialized")

	return client, nil
}
