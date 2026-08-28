package user

import (
	"errors"
	"time"
)

type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Password     string     `json:"password"`
	PasswordHash string     `json:"password_hash"`
	Email        string     `json:"email"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type UserRegisteredEvent struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	UserNotFound     = errors.New("user not found")
	TokenNotFound    = errors.New("token not found")
	TokenBlacklisted = errors.New("token blacklisted")
)
