package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"authe/internal/config"
	packCache "authe/internal/platform/cache"
	"authe/internal/platform/database"
	"authe/internal/platform/security"
	user "authe/internal/user"
	userHandler "authe/internal/user/handler"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		URL:          cfg.DB.URL,
		MaxConns:     cfg.DB.MaxConns,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxConnIdle:  cfg.DB.MaxConnIdle,
	})
	if err != nil {
		slog.Error("failed to init database", "err", err)
		return
	}
	defer db.Close()

	redisCache, err := packCache.New(cfg.Redis.URL)
	if err != nil {
		slog.Error("failed to init redis", "err", err)
		return
	}
	defer redisCache.Close()

	tokenManager := security.NewTokenManager(cfg.Security.JWTSecret, cfg.Security.TokenTTL)

	repository := user.NewRepository(db)
	cache := user.NewCache(redisCache, tokenManager, cfg.Security.RateLimit)
	service := user.NewService(repository, cache, tokenManager)
	handler := userHandler.NewHandler(service)

	mux := http.NewServeMux()
	middleware := userHandler.NewMiddleware(cache, tokenManager, cfg.Security.RateLimit)
	handler.Route(mux, middleware)

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		slog.Info("HTTP Server started")
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "err", err)
		}
		slog.Info("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	slog.Info("Shutting down Postgres connection pool.")
	db.Close()
	slog.Info("Shutting down Redis server.")
	redisCache.Close()

	log.Println("Graceful shutdown complete.")
}
