package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"authe/internal/platform/database"
	user "authe/internal/user"
	userHandler "authe/internal/user/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("failed to load .env")
		return
	}
	dbUrl := os.Getenv("DB_URL")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, dbUrl)
	if err != nil {
		slog.Error("failed to init database", "err", err)
		return
	}

	repository := user.NewRepository(db)
	service := user.NewService(repository)
	handler := userHandler.NewHandler(service)

	mux := http.NewServeMux()
	handler.Route(mux)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		slog.Error("failed to serve", "err", err)
		return
	}
}
