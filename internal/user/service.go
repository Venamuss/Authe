package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"authe/internal/platform/security"
)

type Repository interface {
	Create(context.Context, *User) (int, error)
	Get(context.Context, int) (*User, error)
	Update(context.Context, int, *User) error
	Delete(context.Context, int) error
	GetByUsername(context.Context, string) (*User, error)
}

type Cache interface {
	SaveUser(context.Context, *User, time.Duration) error
	DeleteUser(context.Context, int) error
	GetUserById(context.Context, int) (*User, error)
	GetUserByUsername(context.Context, string) (*User, error)
	BlacklistToken(ctx context.Context, token string, expire time.Duration) error
}

type TokenManager interface {
	CreateToken(username string) (string, error)
	VerifyToken(tokenString string) error
	ExtractClaimsWithMap(tokenString string) (jwt.MapClaims, error)
}

type EventProducer interface {
	SendEvent(ctx context.Context, key string, payload any) error
}

type service struct {
	repo         Repository
	cache        Cache
	tokenManager TokenManager
	producer     EventProducer
}

func NewService(repository Repository, cache Cache, tokenManager TokenManager, producer EventProducer) *service {
	return &service{
		repo:         repository,
		cache:        cache,
		tokenManager: tokenManager,
		producer:     producer,
	}
}

func (s *service) Create(ctx context.Context, user *User) (int, error) {
	passwordHash, err := security.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = passwordHash

	id, err := s.repo.Create(ctx, user)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	event := UserRegisteredEvent{
		UserID:    id,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: time.Now(),
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.producer.SendEvent(bgCtx, fmt.Sprintf("%d", id), event); err != nil {
			slog.Error("failed to send event", "err", err, "id", id)
		}
	}()

	return id, nil
}

func (s *service) Get(ctx context.Context, id int) (*User, error) {
	user, err := s.cache.GetUserById(ctx, id)
	if err != nil {
		user, err = s.repo.Get(ctx, id)

		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		s.cache.SaveUser(ctx, user, time.Minute*15)
		return user, nil
	}

	return user, nil
}

func (s *service) Update(ctx context.Context, id int, user *User) error {
	if user.Password != "" {
		passwordHash, err := security.HashPassword(user.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = passwordHash
	}

	err := s.repo.Update(ctx, id, user)
	if err != nil {
		return err
	}
	err = s.cache.DeleteUser(ctx, id)
	if err != nil {
		slog.Error("failed to delete user from cache", "err", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		slog.Error("failed to delete user", "err", err)
		return err
	}
	err = s.cache.DeleteUser(ctx, id)
	if err != nil {
		slog.Error("failed to delete user from cache", "err", err)
	}
	return nil
}

func (s *service) IsExistByUsername(ctx context.Context, username string) error {
	_, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, UserNotFound) {
			return fmt.Errorf("user not found: %w", err)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	return nil
}

func (s *service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, UserNotFound) {
			return "", fmt.Errorf("user not found: %w", err)
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if !security.CheckPassword(password, user.PasswordHash) {
		return "", fmt.Errorf("invalid password: %w", err)
	}

	token, err := s.tokenManager.CreateToken(username)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return "Bearer " + token, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := s.tokenManager.ExtractClaimsWithMap(token)
	if err != nil {
		return fmt.Errorf("failed to extract claims: %w", err)
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		return fmt.Errorf("failed to get expiration time: %w", err)
	}
	if exp == nil {
		return errors.New("token has no expiration time")
	}

	ttl := time.Until(exp.Time)
	if ttl <= 0 {
		return nil
	}

	err = s.cache.BlacklistToken(ctx, token, ttl)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}
