package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	URL          string
	MaxConns     int32
	MaxIdleConns int32
	MaxConnIdle  time.Duration
}

type Postgres struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*Postgres, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdle
	poolConfig.MaxConnLifetime = cfg.MaxConnIdle

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}

func (post *Postgres) Close() {
	post.Pool.Close()
}
