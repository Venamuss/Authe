package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authe/internal/platform/security"
)

type Repository interface {
	Create(context.Context, *User) (int, error)
	Get(context.Context, int) (*User, error)
	Update(context.Context, int, *User) error
	Delete(context.Context, int) error
	GetByUsername(context.Context, string) (*User, error)
}

type service struct {
	repo Repository
}

func NewService(repository Repository) *service {
	return &service{
		repo: repository,
	}
}

func (s *service) Create(ctx context.Context, user *User) (int, error) {
	passwordHash, err := security.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = passwordHash

	fmt.Println(security.CheckPassword("password", "$2a$10$OvcHP9EZ9bYc4xfKwBPRl./t2kRtNT71q0PZGpFsxLBd.OEgvzFYi"))
	return s.repo.Create(ctx, user)
}

func (s *service) Get(ctx context.Context, id int) (*User, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) Update(ctx context.Context, id int, user *User) error {
	return s.repo.Update(ctx, id, user)
}

func (s *service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) IsExistByUsername(ctx context.Context, username string) error {
	_, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user not found: %w", err)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	return nil
}

func (s *service) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("user not found: %w", err)
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if !security.CheckPassword(password, user.PasswordHash) {
		return "", fmt.Errorf("invalid password: %w", err)
	}

	token, err := security.CreateToken(username)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return "Bearer: " + token, nil
}
