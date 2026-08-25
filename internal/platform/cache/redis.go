package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

func New(connString string) (*Redis, error) {
	options, err := redis.ParseURL(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	client := redis.NewClient(options)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Redis{Client: client}, nil
}

func (r *Redis) Close() {
	r.Client.Close()
}
