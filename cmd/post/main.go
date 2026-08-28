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

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"authe/internal/config"
	"authe/internal/platform/telemetry"
	httpPostHandler "authe/internal/post/handler/http"
	userV1 "authe/pkg/proto/user/v1"
)

func main() {
	cfg := config.MustLoad()

	tp, err := telemetry.InitTracer(context.Background(), "post-service", cfg.Telemetry.JaegerURL)
	if err != nil {
		slog.Error("failed to init tracer", "err", err)
		return
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown tracer", "err", err)
		}
	}()

	grpcPort := strings.TrimPrefix(cfg.GRPC.Port, ":")
	userGRPCAddr := net.JoinHostPort(cfg.GRPC.Addr, grpcPort)

	conn, err := grpc.NewClient(userGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		slog.Error("failed to connect to grpc server", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	userClient := userV1.NewUserServiceClient(conn)

	mux := http.NewServeMux()
	handler := httpPostHandler.NewPostHandler(userClient)
	handler.Route(mux)

	addr := cfg.HTTP.Port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      otelhttp.NewHandler(mux, "post-service"),
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
}
