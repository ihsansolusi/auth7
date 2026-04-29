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

	opt, err := redis.ParseURL(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse redis URL: %w", op, err)
	}
	opt.PoolSize = cfg.PoolSize
	opt.MinIdleConns = cfg.MinIdleConns
	opt.MaxRetries = cfg.MaxRetries
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	logger.Info().
		Str("op", op).
		Int("pool_size", cfg.PoolSize).
		Msg("redis client initialized")

	return client, nil
}
