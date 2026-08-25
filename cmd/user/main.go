package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"authe/internal/config"
	packCache "authe/internal/platform/cache"
	"authe/internal/platform/database"
	"authe/internal/platform/security"
	user "authe/internal/user"
	grpcUserHandler "authe/internal/user/handler/grpc"
	httpUserHandler "authe/internal/user/handler/http"
	userV1 "authe/pkg/proto/user/v1"
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
	cache := user.NewCache(redisCache)
	service := user.NewService(repository, cache, tokenManager)

	validate := validator.New()
	handler := httpUserHandler.NewHandler(service, validate)

	mux := http.NewServeMux()
	middleware := httpUserHandler.NewMiddleware(cache, tokenManager, cfg.Security.RateLimit)
	handler.Route(mux, middleware)

	addr := cfg.HTTP.Port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	grpcPort := cfg.GRPC.Port
	if !strings.HasPrefix(grpcPort, ":") {
		grpcPort = ":" + grpcPort
	}
	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		slog.Error("failed to listen", "err", err)
	}

	grpcServer := grpc.NewServer()
	userV1.RegisterUserServiceServer(grpcServer, grpcUserHandler.NewServer(service, tokenManager))

	reflection.Register(grpcServer)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			slog.Error("failed to serve", "err", err)
		}
	}()

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
	grpcServer.GracefulStop()

	log.Println("Graceful shutdown complete.")
}
