package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	redisCache "authe/internal/platform/cache"
)

type cache struct {
	redis *redisCache.Redis
}

func NewCache(redis *redisCache.Redis, tokenManager TokenManager, rateLimitMax int) *cache {
	return &cache{
		redis: redis,
	}
}

func (c *cache) SaveUser(ctx context.Context, user *User, TTL time.Duration) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	primaryKey := fmt.Sprintf("user:id:%d", user.ID)
	indexKey := fmt.Sprintf("user:username:%s", user.Username)

	pipe := c.redis.Client.TxPipeline()
	pipe.Set(ctx, primaryKey, userJSON, TTL)
	pipe.Set(ctx, indexKey, user.ID, TTL)

	_, err = pipe.Exec(ctx)
	return err
}

func (c *cache) DeleteUser(ctx context.Context, id int) error {
	primaryKey := fmt.Sprintf("user:id:%d", id)

	user, err := c.GetUserById(ctx, id)
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to get user: %w", err)
	}

	pipe := c.redis.Client.TxPipeline()
	pipe.Del(ctx, primaryKey)
	if user != nil && user.Username != "" {
		indexKey := fmt.Sprintf("user:username:%s", user.Username)
		pipe.Del(ctx, indexKey)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (c *cache) DeleteUserByUsername(ctx context.Context, username string) error {
	indexKey := fmt.Sprintf("user:username:%s", username)
	user, err := c.GetUserByUsername(ctx, username)
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to get user: %w", err)
	}
	primaryKey := fmt.Sprintf("user:id:%d", user.ID)

	pipe := c.redis.Client.TxPipeline()
	pipe.Del(ctx, indexKey)
	pipe.Del(ctx, primaryKey)

	_, err = pipe.Exec(ctx)
	return err
}

func (c *cache) GetUserById(ctx context.Context, id int) (*User, error) {
	primaryKey := fmt.Sprintf("user:id:%d", id)

	userJson, err := c.redis.Client.Get(ctx, primaryKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user User

	err = json.Unmarshal([]byte(userJson), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (c *cache) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	indexKey := fmt.Sprintf("user:username:%s", username)

	userId, err := c.redis.Client.Get(ctx, indexKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, UserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	primaryKey := fmt.Sprintf("user:id:%s", userId)
	userJSON, err := c.redis.Client.Get(ctx, primaryKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user User

	err = json.Unmarshal([]byte(userJSON), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (c *cache) BlacklistToken(ctx context.Context, token string, expire time.Duration) error {
	tokenKey := fmt.Sprintf("blacklist:%s", token)

	pipe := c.redis.Client.TxPipeline()
	pipe.Set(ctx, tokenKey, token, expire)

	_, err := pipe.Exec(ctx)
	return err
}

func (c *cache) CheckBlacklistToken(ctx context.Context, token string) error {
	token = strings.TrimPrefix(token, "Bearer ")
	tokenKey := fmt.Sprintf("blacklist:%s", token)

	n, err := c.redis.Client.Exists(ctx, tokenKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if n == 1 {
		return TokenBlacklisted
	}

	return nil
}

func (c *cache) CheckTimeout(ctx context.Context, ip string) (int, error) {
	ipKey := fmt.Sprintf("login_attempts:%s", ip)
	attempts, err := c.redis.Client.Incr(ctx, ipKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to incr login attempts: %w", err)
	}

	if attempts == 1 {
		_ = c.redis.Client.Expire(ctx, ipKey, 5*time.Minute).Err()
	}
	return int(attempts), nil
}
