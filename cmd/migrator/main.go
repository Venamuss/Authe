package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"authe/internal/platform/database"
	"authe/migrations"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env", "err", err)
		return
	}

	dbUrl := os.Getenv("DB_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.New(ctx, dbUrl)
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
