package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"authe/internal/config"
	"authe/internal/user"
)

func main() {
	cfg := config.MustLoad()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Kafka.Brokers,
		Topic:          cfg.Kafka.Topic,
		GroupID:        "notifications-service-group",
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})
	defer reader.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Graceful shutdown initiated.")
		cancel()
		reader.Close()
	}()

	slog.Info("Starting consumer. Notifications service started.")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			slog.Error("failed to fetch message", "err", err)
			continue
		}

		var event user.UserRegisteredEvent

		if err := json.Unmarshal(msg.Value, &event); err != nil {
			slog.Error("failed to unmarshal message", "err", err)
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		slog.Info("[Notification]",
			"user_id", event.UserID,
			"username", event.Username,
			"email", event.Email,
			"created_at", event.CreatedAt)

		if err := reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("failed to commit message", "err", err)
		}
	}

	slog.Info("Notification service stopper gracefully")
}
