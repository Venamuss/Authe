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

var (
	UserNotFound = errors.New("user not found")
)
