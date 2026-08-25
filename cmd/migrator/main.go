package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"authe/internal/config"
	"authe/internal/platform/database"
	"authe/migrations"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.New(ctx, database.Config{
		URL:          cfg.DB.URL,
		MaxConns:     cfg.DB.MaxConns,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxConnIdle:  cfg.DB.MaxConnIdle,
	})

	if err != nil {
		slog.Error("database failed", "err", err)
		return
	}

	db := stdlib.OpenDBFromPool(pool.Pool)

	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect("pgx"); err != nil {
		slog.Error("failed to set dialect", "err", err)
		return
	}

	if err := goose.Up(db, "."); err != nil {
		slog.Error("failed to migrate up", "err", err)
		return
	}

	slog.Info("Successfully migrated up")
}
