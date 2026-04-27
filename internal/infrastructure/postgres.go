package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/ihsansolusi/auth7/pkg/config"
)

func NewPrimaryPool(ctx context.Context, cfg config.DatabasePoolConfig, logger zerolog.Logger) (*pgxpool.Pool, error) {
	const op = "infrastructure.NewPrimaryPool"

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("%s: parse config: %w", op, err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: create pool: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	logger.Info().
		Str("op", op).
		Int("max_conns", int(poolConfig.MaxConns)).
		Msg("primary pool initialized")

	return pool, nil
}

func NewReplicaPool(ctx context.Context, cfg config.ReplicaConfig, logger zerolog.Logger) (*pgxpool.Pool, error) {
	const op = "infrastructure.NewReplicaPool"

	if !cfg.Enabled || cfg.DSN == "" {
		logger.Info().Str("op", op).Msg("replica disabled")
		return nil, nil
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("%s: parse config: %w", op, err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: create pool: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	logger.Info().
		Str("op", op).
		Msg("replica pool initialized")

	return pool, nil
}
