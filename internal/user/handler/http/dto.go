package user

import (
	"time"

	domain "authe/internal/user"
)

// --- Входящие DTO (Requests) ---

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty" `
	Password *string `json:"password,omitempty"`
}

// --- Исходящие DTO (Responses) ---

type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// --- Мапперы (Конвертеры) ---

func (r *CreateUserRequest) ToDomain() *domain.User {
	return &domain.User{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
	}
}

func (r *UpdateUserRequest) ToDomain() *domain.User {
	u := &domain.User{}
	if r.Username != nil {
		u.Username = *r.Username
	}
	if r.Email != nil {
		u.Email = *r.Email
	}
	if r.Password != nil {
		u.Password = *r.Password
	}
	return u
}

func ToUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
